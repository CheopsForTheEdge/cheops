# cheops

Generic prototype to manage replication of resources


# Compiling and running on Goland

First of all, you *don't have to* use Goland, but this is just the way we 
will describe:
- Download Goland [here](https://www.jetbrains.com/go/) - you can have a 
  professional edition with your student account -
- Git clone the project then import it in the IDE
- Run the project. 
- Cf [tests README](tests/README.md) for testing


# Global working principles

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

In more details:

```
.
├── .gitignore             # Specify which file to ignore in git versioning
├── README.md              # This file
├── todo.org               # What remains to do
├── cheops                 # Where replication is done
│   ├── api                # API package
│   │   └── api.go         # Defines the router and routes
│   ├── database           # Database package
│   │   └── database.go    # Glue to the database
│   ├── endpoint           # Endpoint package
│   │   └── endpoint.go    # Manage different endpoints
│   ├── main               # Main package
│   │   └── main.go        # Only the main function
│   ├── replication        # Replication package
│   │   └── replication.go # Replicants and replicas management
│   ├── request            # Request package
│   │   └── request.go     # Get the request and transfer it to the driver
├── drivers
│   └── googleboutique    # Driver for Google Boutique
│   └── k8s               # Driver for Kubernetes
│   └── openstack         # Driver for OpenStack
│   │   ├── openstack.go  # Interpreter for the request
│   │   └── scope.go      # Interprets the scope (might inherit from a scope)
├── tests                 # Where are the tests
│   ├── .gitignore        # Gitignore for the tests
│   ├── README.md         # README for the tests
│   ├── requirements.txt  # Requirements file for the tests (cf README)
│   └──testingAPI.yaml    # API tests
```

## Functioning

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
To go *beyond the clouds* and to also stay close to the scope, I chose the 
name [Cheops](https://www.esa.int/Science_Exploration/Space_Science/Cheops).

[Cure]: https://pages.lip6.fr/Marc.Shapiro/papers/Cure-final-ICDCS16.pdf
[Sagas]: http://www.amundsen.com/downloads/sagas.pdf
[envoy]: https://www.consul.io/docs/connect/proxies/envoy
[Golang conventions]: (https://medium.com/@kdnotes/golang-naming-rules-and-conventions-8efeecd23b68)