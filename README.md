# Cheops

Cheops turns a monolithich application into a geo-distributed service
automatically synchronized and redundant by replicating all user operations to
eventually converge to the same state. It is assumed that operations are always
associated to a specific resource. Cheops gives the possibility to specify the
exact distribution of each resource manually so that operators can define how
they want them to be spread.

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
a synchronous process.

Each resource has its own set of nodes: the replication will only distribute
operations to the node that are responsible for the resource. The diagram above
shows that all node can talk to all other nodes, but only the information
related to a specific node are sent to it.

The data model is defined in model/ : it might make sense to explore it to
understand the following sections

## Cheops

Here's a schema showing more details about how Cheops is organized:

![Cheops](./cheops-2.svg)

There are 3 main parts: the API, the Replicator and the Backends

### API

This is where the http api is defined. Users can call it directly, or through a CLI.

To send a request, users send a POST to /exec/{id} where id is the resource
identifier. The body is a multipart form with the following parts:
- sites: a list of all the desired locations of the resources, separated with a
"&"
- command: the command to run
- type: the type of the command, to configure consistency classes. See the
related paragraph in CONSISTENCY.md
TODO
- config: the resource configuration
- files: all the files necessary for the command to properly execute

This bundle is transformed into an operation struct and sent to the Replicator layer

TODO: /show /show_local

### Replicator
TODO: explain resource config

### Backend

This is the simplest of the layers. It is called with a list of commands to run and runs them. At the moment the handling is hardcoded to execute shell commands (hence why there is a "command" field in the input). This is how the genericity is provided: to run commands for other applications, the command itself manages the backend to use.

## Replication

The replication is provided by [CouchDB](https://couchdb.apache.org/), a
reliable, efficient synchronization database. It is a kv store associating
strings to JSON documents. In order to implement reliable synchronization
without losing data it also associates each modification of a document with a
revision string, and to make a change the user must also give the existing
revision they want to change. This means the representation of the lifecycle of
a given key is not a list of versions, but a graph. More details are given in
the documentation at https://docs.couchdb.org/en/stable/intro/overview.html.

Each JSON document contains the name of the resource, the list of operations
that are to be applied and the locations where the resource is expected to live.
When a new document arrives in CouchDB, a process in Cheops will see the new
document and create, if they don't already exist, a replication from the local
node to each of the locations in the new document (except of course to itself).
It is possible that such a replication (for example, from node23 to node47) was
already created for another resource; we effectively decorrelate those
replication jobs from the resources themselves and only look at the locations.
See replicator/replicator.go:/func.*replicate for the implementation.

As a reminder, replication will make sure that all versions of all nodes are
known from every node; there can be a conflict, typically when the same
resource is updated from 2 different places before replication converged. This
situation is described, and the solution explained, in CONSISTENCY.md. To see
how it is done in the code, see
[replicator/replicator.go](replicator/replicator.go:/func resolveMerge)

## Configuration and usage

At the moment Cheops has deployment scripts to be used in Grid5000 only: see
the tests/g5k folder to understand how it is deployed and reuse it in other
settings.


## Files

```
.
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