# Cheops

Cheops turns a monolitic application into a geo-distributed service
automatically synchronized and redundant by replicating all user operations to
eventually converge to the same state. It is assumed that operations are always
associated to a specific resource. Cheops gives the possibility to specify the
exact distribution of each resource manually so that operators can define how
they want them to be spread.

The repo is being maintained with Acme, and as such some Acme-specific
artifacts will exist:
- a guide file is used with common helpers
- files are not linked to just by name, they also contain a string to search
for that points to the exact information, regardless of changes

# License

[GNU Affero General Public License](https://www.gnu.org/licenses/agpl-3.0.html#license-text)

See the [LICENSE](./LICENSE) document for the full text


# Architecture

Here's a top-level view of how Cheops is built

![Architecture](./cheops-1.svg)

Cheops organizes independent nodes with fluctuant networking and availability
conditions, represented by the boxes above. Each node has 3 elements: a CouchDB
instance, the application to geo-distribute, and the Cheops process itself
sitting in the middle.

Users always interact directly with Cheops. When a request is made to an
application through the CLI, for a specific resource, cheops captures it and
all the required files and stores everything in CouchDB. CouchDB then
synchronizes that request with all other CouchDB instances in a flooding manner
(such that if a node is down, the message can still be propagated). Multiple
users may interact in parallel on multiple nodes for the same resource: all
that information is stored and propagated.

When CouchDB receives a new request it informs the Cheops process of that
changes. Cheops will then manage conflict in operations thanks to its internal
algorithms and produce a set of operations to be run: Cheops then runs those
operations and stores the result back in CouchDB. The same replication
algorithm is responsible for propagating all results to all the nodes such that
the user can see it from everywhere, especially in the common case from the
node where the request was initially made. The CLI actually hides this process
by waiting up to 30 seconds for all replies to appear, giving the appearance of
a synchronous process, and returns the result of the executions to the HTTP
caller. If all nodes haven't synced back their execution status within this
timeout, the initial HTTP caller will get the reply from those who have replied
only, but the process still happens in the background. Cheops, through CouchDB,
will continuously try to synchronize and run operations everywhere it is needed.

Each resource has its own set of nodes: the replication will only distribute
operations to the node that are responsible for the resource. The diagram above
shows that all node can talk to all other nodes, but only the information
related to a specific node are sent to it.

The data model is defined in [model/](model/) : it might make sense to explore
it to understand the following sections.

Requests are always shell commands to be executed in a standard Debian
environment right now. The application is expected to be usable through such
commands. The shell commands can be anything that will run on the nodes and
will be executed by the Cheops process: as such, no security is in place and
the user has the same rights as the Cheops process. It is possible to send
files along with the command if those files are needed for the command to
execute: they will be stored along the command, be replicated together, etc...

Cheops is written in Go because it is a solid production-ready language with
good primitives for concurrent jobs and synchronization work.

## Cheops

Here's a schema showing more details about how Cheops is organized:

![Cheops](./cheops-2.svg)

There are 3 main parts: the API, the Replicator and the Backends

### API

This is where the http api is defined. Users can call it directly, or through a
CLI.

To send a request, users send a POST to `/exec/{id}` where id is the resource
identifier. The body is a multipart form with the following parts:
- sites: a list of all the desired locations of the resource, separated with a
		"&"
- command: the command to run
- type: the type of the command, to configure consistency classes. See the
		related paragraph in [CONSISTENCY.md](CONSISTENCY.md)
- config: the resource configuration
- files: all the files necessary for the command to properly execute

This bundle is transformed into an operation struct and sent to the Replicator
layer.

The `/show/{id}` endpoint is used to represent a resource in a user-defined
way, and
gather that representation from all available nodes. The operation is
broadcasted to all reachable nodes, meaning that the view is the most recent
available. It is not part of the registered operations and doesn't go through
the usual syncing and merging flow. The body must be a multipart form with the
following fields:
- sites: the list of all the locations of the resource, separated with a
"&"
- command: the command to run to represent the resource (eg `cat XXX` for a
file)

This call will actually call `/show_local/{id}` endpoint on all the sites
(including locally). That second call executes the command and returns the
standard output.

The API is an HTTP API and there are cli helpers in the folder of the same
name. The cli can be used to make it simpler to use cheops. Here's an example
usage:

```sh
%  cheops \
    --id deployment-id \
    --sites ’site1&site2&site3’ \
    --command "sudo kubectl create deployment -f {recipe.yml}" \
    --config config.json \
    --type create
```

The arguments `id`, `sites`, `config` and `type` are the same as explained
above. `command` is the same except it can contain files between `{` and `}`:
those files are taken from the computer where the cli is run (ie the computer
of the user) and sent as files as described above.

### Replicator
This is the main part of the cheops application. It is responsible for merging
and asking the backend to run operations, to present a pseudo-synchronous
interface to callers, and to configure CouchDB for replication. It takes input
from the api layer that it transforms into json documents: the model is in
[model/crdt_document.go](model/crdt_document.go:/type ResourceDocument). The
files are base64-encoded and recorded along with the command so it can be
re-run anytime.

The Replicator layer creates an implementation of an RCB by creating a directed
 acyclic graph with operations as nodes and causality as edges. An edge between
two operations is always directed from the operation happening before to the
operation happening after. As such, 2 operations may not have a direct edge,
but still be indirectly related through parentage: if `operationA` is followed
causally by `operationB` which is itself followed causally by `operationC` then
we know there is a causal link between `operationA` and `operationC`. A
conflict is identified when two operations have a common ancestor but there is
no path between the two. When that happens, it is resolved using the operations
relationship matrix for a resource. As an optimization, we realized that some
operations are replacing the entire state of the resource: causally, it is not
necessary to remember what happened before because it will be overwritten.
Since the graph is cut every time a replace operation happens, it is enough to
remember all operations in a single CouchDB document and let CouchDB alert us
in case of conflict. In that case a conflict is solved by looking only at the
first operation (since the following ones are iterative, they should all be
played in the same order). Based on this simplification, upon receiving an
operation from the API layer the Replication layer generates the new list of
operations (either by appending or by replacing as seen above).

All the details about the exact consistency details are explained in
[CONSISTENCY.md](CONSISTENCY.md). We will only discuss the implementation
details here. As explained in that document, it is necessary for Cheops to know
how multiple requests will interoperate. This means that the operator must
define relationships between operations: this is defined in a Relationship
Matrix inside the ResourceConfig file.

The ResourceConfig can be sent on each push of a new operation, but right now
it is expected to be sent at the beginning of the life of a resource (ie when
it is first created). When ResourceConfig files are pushed later, concurrently
or not, CouchDB will give us a consistent winner thanks to its LWW algorithm.
Because we do not know which one will be taken, it is impossible to know for
sure what will be the result; to be sure, the only way is by sending a new
ResourceConfig at the end. Modifying ResourceConfig is therefore not
recommended at the time.

Inside the ResourceConfig the major structure is the RelationshipMatrix. It has
more details in the paper (see the [Thesis Manuscript,
2023](https://theses.hal.science/tel-04081084/)) but the gist is the following:
the matrix is a list of tuples with the following fields:
- Before: a type of operation
- After: a type of operation
- Result: the interaction between the two types.

The `type` here is actually a simple string that is used to identify
operations: it is similar to a "tag". Each operation, when it is pushed, is
given such a tag, and the matrix will say how operations collaborate. For
example a SET operation in redis might be given a "set" type to be easily
identifiable.

`Before` and `After` are thus two types. The operator is expected to describe
how the two will be handled when they are met in that order. This information
is needed in 2 cases:
- when a new operation is given, to see whether it can be added on top of the
existing ones or a new block must be created
- when two conflicting list of operations exist, the first operation on each
side is compared

As alluded above, and described in [CONSISTENCY.md](CONSISTENCY.md) in further
details, the list of operations might be periodically "pruned" when operations
with the proper types are added to the existing list.

An important consequence of the design is that until any pruning happens, the
entirety of the payload is stored along operations, in CouchDB. Typically this
payload might also exist in the application and as such be stored twice: this
is highly inefficient and is an avenue for potential optimization in the
future. See Areas of Improvement for more information about what can be done
about that.

### Backend

This is the simplest of the layers. It is called with a list of commands to run
and runs them. At the moment the handling is hardcoded to execute shell
commands (hence why there is a "command" field in the input). This is how the
genericity is provided: to run commands for other applications, the command
itself manages the backend to use.

## Replication

The replication is provided by [CouchDB](https://couchdb.apache.org/), a
reliable, efficient synchronization database. It is a kv store associating
strings to JSON documents. In order to implement reliable synchronization
without losing data it also associates each modification of a document with a
revision string, and to make a change the user must also give the existing
revision they want to change. This means the representation of the lifecycle of
a given key is not a list of versions, but a graph. More details are given in
[the CouchDB documentation](https://docs.couchdb.org/en/stable/intro/overview.html).
CouchDB has a concept of "databases", which are simply collections of json
documents; each "database" can be useful if different access rights are needed,
but for Cheops we only use a single "database" called "cheops" where the
credentials are admin/password.

Each JSON document contains the name of the resource, the list of operations
that are to be applied and the locations where the resource is expected to live.
When a new document arrives in CouchDB, a process in Cheops will see the new
document and create, if they don't already exist, a replication from the local
node to each of the locations in the new document (except of course to itself).
It is possible that such a replication (for example, from node23 to node47) was
already created for another resource; we effectively decorrelate those
replication jobs from the resources themselves and only look at the locations.
See [replicator/replicator.go](replicator/replicator.go:/func.*replicate) for
the implementation.

As a reminder, replication will make sure that all versions of all nodes are
known from every node; there can be a conflict, typically when the same
resource is updated from 2 different places before replication converged. This
situation is described, and the solution explained, in
[CONSISTENCY.md](CONSISTENCY.md). To see how it is done in the code, see
[replicator/replicator.go](replicator/replicator.go:/func resolveMerge)

## Configuration and usage

At the moment Cheops has deployment scripts to be used in Grid5000 only: see
the [tests/g5k](tests/g5k) folder to understand how it is deployed and reuse
it in other settings.

Note that it is crucial to define your resource configuration, especially the
relationship matrix, for cheops to work properly. See how to do that in the
multiple tests.

## Files

```
.
├── guide               # Acme-specific utilities
├── api                	# Defines the routes for handling requests
├── replicator        	# Replicator package responsible for syncing
├── backends        	# Backends package responsible for executing requests
├── model        		# The data model
├── chephren			# Defines the routes for the ui
├── chephren-ui			# The ui files
├── cli        			# All the command-line interface blobs
├── tests/g5k           # infrastructure for testing on g5k, see tests/g5k/README.md
├── CONSISTENCY.md 		# How Cheops ensures a consistent state on all nodes
└── README.md           # This file


```

## Areas of improvement

Here are some of the pain points related to how Cheops works, either because of
its architecture or because of its first-implementation status, and where
future development may propose a boost in performances and usability.

### Storing data only once

Imagine an operation saying "insert this image in the filesystem". This
operation must maintain the whole image in CouchDB for the future, but the
image will also be stored in the filesystem as this is the meaning of the
operation.

To reduce this duplication, multiple strategies can be used:
- if the application stores its data in a deduplicating filesystem, CouchDB can
be configured to use the same filesystem (for example a shared ZFS filesystem
on a cluster).
- the other way around: if the application can choose its storage space,
CouchDB can serve the file thanks to its HTTP endpoints. The caveat is that
CouchDB becomes a strong requirement of Cheops, whereas today it is only an
intermediary for sync.


### Pruning operations when all nodes agree

The composition of a cluster for a given resource is known and cannot change.
Moreover, every node knows the execution status of all operations from other
nodes. Thanks to this every node can know when operations have been properly
executed on all other nodes: it is thus acceptable, if they have been properly
run, to remove them from the list of all operations.

Note that while it is correct from the point of view of operations, the
resource might still diverge in the application. Imagine an operation that
compresses some common file: if the exact compression details are not the same,
the result will differ. It is important for operators to keep a history of
operations to understand why they end up with different objects. From a first
approximation though it is ok to prune operations if they are deterministic.

### Hooks

Cheops has an optimistic 30-second window during which it hopes to have
synchronized and executed the operations on all nodes, but by the very nature
of our work some operation might not have been synchronized to other nodes
(because they or the network is down, typically). This doesn't prevent Cheops
from working in the background continuously: if the network is back up 24 hours
after the operation has been inserted, it will be synchronized, run, and the
result will be synchronized back everywhere.

There is no way for the operator to know about this: outside of the 30-second
window all operations happen in the background. An operator might still be
interested in knowing when something happens: either when a specific operation
was run, or more generally when a resource has been modified. It is possible to
extend Cheops to offer a system of hooks for this.

The simplest way is to plug into CouchDB. It has the `_changes` endpoint
facilitating realtime following of changes (this is what Cheops itself uses) as
described in its
[documentation](https://docs.couchdb.org/en/stable/api/database/changes.html).
However this again puts CouchDB as a fundamental brick of the solution and
prevents any change in that direction. It might be more interesting to offer a
simplified changes feed at the Cheops level (maybe `/changes/{id}` to follow
changes of a specific resource), and tell cheops to follow changes inside
CouchDB for that hypothetical new endpoint. It wouldn't take more than 2 days
for an experienced engineer to build this.

This solution is simplest but requires the operator's computer to always be
turned on: since it is a pull-based mechanism, something needs to continuously
(try to) pull. A more involved mechanism would be push-based, controlled by
Cheops itself:
- sending an email by connecting to a preconfigured smtp server with a specific
account
- sending a message on an irc server
- ping a webhook
- run any kind of command for any scenario (push a log in a supervision system,
...)

Webhooks are often available in current messaging tools and can be a good way
to inform a team of operators, where they usually discuss, that something
happened on a resource.

The potential issue with this approach is that all Cheops node are independent,
and those mechanisms will run independently on each node. If the operator
wishes to know whenever something happens on any node, that is fine, but if
they only want a summary information when all nodes have run the operation some
coordination will be required. An experienced engineer wouldn't take more than
3 days to build the first version, but it could take a week to devise and
implement a "summary" version where only one action is taken when all the nodes
have run the same operation on the same resource.

### More than shell commands

At the time of writing, Cheops operations are always shell commands. This was
chosen for the purpose of experimenting: applications were chose such that they
were usable through shell commands. Thanks to that we elude the particulars of
each application's potential protocol (be it HTTP, custom like Redis, or the
filesystem), because they're not the most relevant aspect of our research.

This might not be a desirable end goal though. Shell commands carry with them
the issue of being security holes: if the user can send anything, they have to
be completely trusted. It also assumes an application and its resources can be
manipulated with shell commands. The shell command also needs to be installed
on the Cheops node.

Because of this it might be interesting to give other options to operators. If
the application is primarily used through HTTP, the shell command we'd use
would be `curl`, but it would be easier and more robust to store the exact HTTP
request as operations. The go language has all the facilities to do that: the
most difficult job will be for future developers to evaluate how to best input
that request to Cheops. Perhaps inspiration can be taken from the
[httpie](https://httpie.io/cli) cli tool, and an equivalent can be built for
Cheops. An extension to `httpie` to include the cheops-specific information,
for example through environment variable, would take a week and allow users to
specify commands like so:

```sh
% ID=my-resource SITES=site1&site2&site3 cheops-web POST /app/api/modify resource=my-resource foo=bar
```

The same idea can be re-appropriated for non-HTTP protocols:
```sh
% ID=my-list SITES=site1&site2&site3 cheops-redis LSET my-list 23 new-val
```

In order to do this, a plugin architecture can be used where all backends are implementation of the same interface:

```go
type Backend interface {
	Run(cmd Command) Result
}
```
and all backends would register themselves at the beginning of the lifecycle of Cheops, that would then pick the proper `Backend` to run the operation with based on a command type.

Such a change would probably take a week to implement for the first protocol, and then a few days for each new protocol

## How to contribute

- Please follow usual [Golang conventions]. If you find some infractions,
  please report (or edit) them.
- When adding, removing or change the use of a file, please change the
  corresponding entry in the README.md.
- Don't hesitate to report an issue!
- Thanks for any contribution :)

## Some other sources

- https://www.nicolasmerouze.com/middlewares-golang-best-practices-examples
- https://medium.com/the-andela-way/build-a-restful-json-api-with-golang-85a83420c9da
- https://golang.org/doc/effective_go

## About the name "Cheops"

This project has been envisionned in the context of the [Discovery
Initiative](https://beyondtheclouds.github.io/).
To go *beyond the clouds* and to also stay close to the scope, the
name [Cheops](https://cheops.unibe.ch/) was chosen.

[Cure]: https://pages.lip6.fr/Marc.Shapiro/papers/Cure-final-ICDCS16.pdf
[Sagas]: http://www.amundsen.com/downloads/sagas.pdf
[envoy]: https://www.consul.io/docs/connect/proxies/envoy
[Golang conventions]: (https://medium.com/@kdnotes/golang-naming-rules-and-conventions-8efeecd23b68)

# History

Cheops is a follow-up from the original PoC for scope-lang's sharing,
which can be found [here](https://github.com/BeyondTheClouds/openstackoid).
The goal of Cheops is to make it more generic, and with more
functionalities.

Cheops work is supported by [Inria](https://www.inria.fr/),
[IMT Atlantique](https://www.imt-atlantique.fr/) and
[Orange Labs](https://www.orange.com/).

[Cheops OpenInfra 2022 Video presentation](https://www.youtube.com/watch?app=desktop&v=7EZ63DMRJhc)

# Publications

- Euro-Par 2021
  - [Preprint](https://hal.inria.fr/hal-03212421v1): Ronan-Alexandre Cherrueau, Marie Delavergne, Adrien Lebre. Geo-Distribute Cloud Applications at the Edge. EURO-PAR 2021 - 27th International European Conference on Parallel and Distributed Computing, Aug 2021, Lisbon, Portugal. pp.1-14. ⟨hal-03212421⟩
  - [Article](https://doi.org/10.1007/978-3-030-85665-6_19): Cherrueau, RA., Delavergne, M., Lèbre, A. (2021). Geo-distribute Cloud Applications at the Edge. In: Sousa, L., Roma, N., Tomás, P. (eds) Euro-Par 2021: Parallel Processing. Euro-Par 2021. Lecture Notes in Computer Science(), vol 12820. Springer, Cham. https://doi.org/10.1007/978-3-030-85665-6_19

- AMP 2021
  - [Preprint](https://hal.inria.fr/hal-03282425)
  - [Article](https://amp.fe.up.pt/2021/papers/paper1/): Marie Delavergne, Ronan-Alexandre Cherrueau, Adrien Lebre. A service mesh for collaboration between geo-distributed services: the replication case. Workshop AMP 2021 colocated with XP 2021 conference, Jun 2021, Online, France. ⟨hal-03282425⟩

- [Slides AMP 2021](https://docs.google.com/presentation/d/1ZusGXEKPaRXQUaodkuvzJ5awdUmU6o8muxNYB-GZOPo/edit?usp=sharing)

- OpenInfra Summit 2022 - Can a "service mesh" be the right solution for the Edge?
  - [Slides](https://gitlab.inria.fr/discovery/cheops/-/raw/master/Infos/Slides_OpenInfra_Summit_2022.pdf)
  - [Video](https://www.youtube.com/watch?v=7EZ63DMRJhc)

- [Poster Compas 2022](https://gitlab.inria.fr/discovery/cheops/-/raw/master/Infos/Poster_Compas_2022.pdf)

- [ICSOC 2022](https://link.springer.com/chapter/10.1007/978-3-031-20984-0_37): Marie Delavergne, Geo Johns Antony, and Adrien Lebre. "Cheops, a service to blow away Cloud applications to the Edge." Service-Oriented Computing: 20th International Conference, ICSOC 2022, Seville, Spain, November 29–December 2, 2022, Proceedings. Cham: Springer Nature Switzerland, 2022.
  - [Research Report](https://hal.inria.fr/view/index/identifiant/hal-03770492): Marie Delavergne, Geo Johns Antony, Adrien Lebre. Cheops, a service to blow away Cloud applications to the Edge. [Research Report] RR-9486, Inria Rennes - Bretagne Atlantique. 2022, pp.1-16. ⟨hal-03770492v2⟩

- [Thesis Manuscript, 2023](https://theses.hal.science/tel-04081084/): Marie Delavergne. Cheops, a service-mesh to geo-distribute micro-service applications at the Edge. Distributed, Parallel, and Cluster Computing [cs.DC]. Ecole nationale supérieure Mines-Télécom Atlantique, 2023. English. ⟨NNT : 2023IMTA0347⟩. ⟨tel-04081084⟩

- [ICFEC 2024]
  - [Preprint](https://inria.hal.science/hal-04522961): Geo Johns Antony, Marie Delavergne, Adrien Lebre, Matthieu Rakotojaona Rainimangavelo. Thinking out of replication for geo-distributing applications: the sharding case. ICFEC 2024 - 8th IEEE International Conference on Fog and Edge Computing, May 2024, Philadelphia, United States. pp.1-8. ⟨hal-04522961⟩

# Chephren

Chephren is a project to build a nice web ui on top of Cheops. It is available here:

    https://gitlab.imt-atlantique.fr/chephren/

To use it, clone the chephren repo, run the build (`npm run build`) and copy
the dist folder into the chephren-ui folder.

Unfortunately the latest version is not up-to-date with the current version of
cheops: it needs to be updated with the latest data model.

TODO: more details on how it works and how to update it