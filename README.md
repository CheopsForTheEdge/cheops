# Cheops

Generic service to manage geo-distributed resources.

Cheops is a follow-up from the original PoC for scope-lang's sharing,
which can be found [here](https://github.com/BeyondTheClouds/openstackoid).
The goal of Cheops is to make it more generic, and with more
functionalities.

Cheops work is supported by [Inria](https://www.inria.fr/),
[IMT Atlantique](https://www.imt-atlantique.fr/) and
[Orange Labs](https://www.orange.com/).

[Cheops OpenInfra 2022 Video presentation](https://www.youtube.com/watch?app=desktop&v=7EZ63DMRJhc)

### Update

This research prototype is currently under heavy development. If you are interested to know more about this project, please contact us.

# Publications

- Euro-Par 2021
  - [Preprint](https://hal.inria.fr/hal-03212421v1): Ronan-Alexandre Cherrueau, Marie Delavergne, Adrien Lebre. Geo-Distribute Cloud Applications at the Edge. EURO-PAR 2021 - 27th International European Conference on Parallel and Distributed Computing, Aug 2021, Lisbon, Portugal. pp.1-14. ⟨hal-03212421⟩
  - [Article](https://doi.org/10.1007/978-3-030-85665-6_19): Cherrueau, RA., Delavergne, M., Lèbre, A. (2021). Geo-distribute Cloud Applications at the Edge. In: Sousa, L., Roma, N., Tomás, P. (eds) Euro-Par 2021: Parallel Processing. Euro-Par 2021. Lecture Notes in Computer Science(), vol 12820. Springer, Cham. https://doi.org/10.1007/978-3-030-85665-6_19

- AMP 2021
  - [Preprint](https://hal.inria.fr/hal-03282425)
  - [Article](https://amp.fe.up.pt/2021/papers/paper1/): Marie Delavergne, Ronan-Alexandre Cherrueau, Adrien Lebre. A service mesh for collaboration between geo-distributed services: the replication case. Workshop AMP 2021 colocated with XP 2021 conference, Jun 2021, Online, France. ⟨hal-03282425⟩

- [Slides AMP 2021](https://docs.google.com/presentation/d/1ZusGXEKPaRXQUaodkuvzJ5awdUmU6o8muxNYB-GZOPo/edit?usp=sharing)

- [Slides OpenInfra Summit 2022](https://gitlab.inria.fr/discovery/cheops/-/raw/master/Infos/Slides_OpenInfra_Summit_2022.pdf)

- [Poster Compas 2022](https://gitlab.inria.fr/discovery/cheops/-/raw/master/Infos/Poster_Compas_2022.pdf)

- [ICSOC 2022](https://link.springer.com/chapter/10.1007/978-3-031-20984-0_37) : Marie Delavergne, Geo Johns Antony, and Adrien Lebre. "Cheops, a service to blow away Cloud applications to the Edge." Service-Oriented Computing: 20th International Conference, ICSOC 2022, Seville, Spain, November 29–December 2, 2022, Proceedings. Cham: Springer Nature Switzerland, 2022.
  - [Research Report](https://hal.inria.fr/view/index/identifiant/hal-03770492): Marie Delavergne, Geo Johns Antony, Adrien Lebre. Cheops, a service to blow away Cloud applications to the Edge. [Research Report] RR-9486, Inria Rennes - Bretagne Atlantique. 2022, pp.1-16. ⟨hal-03770492v2⟩

# License

[GNU Affero General Public License](https://www.gnu.org/licenses/agpl-3.0.html#license-text)

See the [LICENSE](./LICENSE) document for the full text

# Install

```
# Install and run Couchdb
sh -x install-run-couchdb.sh

# Install and run kube
sh -x install-run-kube.sh

# Get go 1.19
apt install golang-1.19
```

# Run

```
rm cheops.log 2> /dev/null
killall cheops.com 2> /dev/null
/usr/lib/go-1.19/bin/go build
kubectl delete all --all
MYFQDN=<my.fq.dn> ./cheops.com 2> cheops.log &
```

# In practice

If you can, deploy some nodes in grid5000 and use the jupyter notebook in `tests/g5k`. There are instructions there about what can be used and done

# Global working principles

TODO: update

Cheops is designed in a P2P manner considering each resource as a black box:
  + Uses scope-lang to define where resources will be replicated and uses
    the forwarding operation.
  + Agents are located on each site
  + Uses heartbeat to check if sites are up and in the network
  + Uses a geo-distributed like database to have resource information only
    where relevant
  + Will provide different level of consistency, the 2 lasts requiring
    *transactionable resources* (having the ability to rollback any
    operation done on them):
    - **"none"**: No guarantees (operations are triggered and that's all).
    - **eventual**: every operation on a replica will be applied to the others
      eventually (will be the focus for now).
    - **transactional eventual**: either with two phases commit
      or long-lived transactions, depending on the resources
      involved. Ensures transactions while still being available. cf [Cure]
      and [Sagas].
    - **strong serializable** : strongest consistency, but the system might
      be unavailable a lot.


```
+---------------------+                          +---------------------+
|                     |                          |                     |
|  Cheops agent 1     +<------------------------>+  Cheops agent 2     |
|                     |                          |                     |
|                 +---+--+                    +--+---+                 |
|                 | DB   |                    | DB   |                 |
+-------+--+------+      +<------------------>+      +---+----+--------+
        ^  |      +------+                    +------+   |    ^
        |  |                                             |    |
        |  |  +------------------------------------------+    |
        |  |  |                                               |
        |  +-----------------------------------------------v  |
        |     |                                            |  |
        v     v                                            v  v
   +----+-----+------+                               +-----+--+--------+
   |                 |                               |                 |
   |                 |                               |                 |
   |   Service A1    |                               |   Service A2    |
   |                 |                               |                 |
   |                 |                               |                 |
   +-----------------+                               +-----------------+
```


# Architecture

TODO: update

In more details:

```
.
├── api                # API package
│   └── api.go         # Defines the router and routes
├── broker
│   ├── broker_receive.go
│   ├── dep.json
│   ├── deployment.json
│   └── go.mod
├── client
│   ├── deployment.json
│   └── server.go
├── endpoint           # Endpoint package
│   ├── endpoint.go    # Manage different endpoints (services, related to the application)
│   └── site.go        # Manage different sites (cheops to cheops)
├── replication        # Replication package
│   └── replication.go # Replicants and replicas management
├── request            # Request package
│   └── request.go     # Get the request and transfer it to the driver
├── glue
│   ├── k8s            # Driver for Kubernetes
│   │   ├── fetch.go   # Interpreter for the request
│   │   └── go.mod
│   └── openstack      # Driver for OpenStack
│       ├── fetch.go   #
│       └── go.mod     #
├── infos
│   ├── Poster_Compas_2022.pdf
│   └── Slides_OpenInfra_Summit_2022.pdf
├── operation
│   ├── operation.go
│   └── replication.go
├── request
│   ├── request.go
│   └── scope.go
├── tests                 # Where are the tests
│   ├── .gitignore        # Gitignore for the tests
│   ├── README.md         # README for the tests
│   ├── requirements.txt  # Requirements file for the tests (cf README)
│   ├── testingAPI.yaml   # API tests
│   ├── serviceA          # Mock service to test Cheops
│   └── serviceB          # Mock service to test Cheops
├── tutorials
├── utils                 # Where are the tests
│   ├── config.go
│   ├── database.go
│   ├── launch_db.sh
│   └── utils.go
├── .gitignore             # Specify which file to ignore in git versioning
├── config.yml
├── dockerfile
├── go.mod
├── install.sh
├── k8s_debian.sh
├── main.go
├── README.md              # This file


```

## Functioning

TODO: update

When a creation request is made to an application, [envoy] captures it and
transfers
it to Cheops. Cheops uses its list of drivers to check which one to use. The
request is read by the driver, and the scope is extracted and sent to the
module in charge of location interpretation.

A replicant is created with a metaID and added to the database (for now, a
KV store).
In parallel, the request is separated in as many local requests as necessary
and sent to the Cheops of involved sites, with the replicant to add in their
database.

Every Cheops involved sends the request back to the right service locally
and adds the replicant to its database. When receiving the response, they
add the local ID to the replicant and transfer the information to all
involved Cheops again.

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

### About the name "Cheops"

This project has been envisionned in the context of the [Discovery
Initiative](https://beyondtheclouds.github.io/).
To go *beyond the clouds* and to also stay close to the scope, the
name [Cheops](https://cheops.unibe.ch/) was chosen.

[Cure]: https://pages.lip6.fr/Marc.Shapiro/papers/Cure-final-ICDCS16.pdf
[Sagas]: http://www.amundsen.com/downloads/sagas.pdf
[envoy]: https://www.consul.io/docs/connect/proxies/envoy
[Golang conventions]: (https://medium.com/@kdnotes/golang-naming-rules-and-conventions-8efeecd23b68)
