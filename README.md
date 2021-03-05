# cheops

Generic prototype to manage replication of resources



# Compiling and running on Intellij IDE
Git clone the project then import it from the IDE (project type = maven).

pom.xml contains all project's dependencies. You need to download them first.

For this, open pom.xml, right mouse-click, select maven then reload project.

Build the project then run it. There must be a local etcd running.

Download [etcd binary](https://github.com/etcd-io/etcd/releases) then run it.


# Global working principles

Cheops is designed in a P2P manner considering each resource as a black box.
  + Agents are located on each site
  + Uses heartbeat to check if sites are up and in the network
  + Uses a geo-distributed like database to have resource information only where relevant


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


# architecture


```
+------------+
|            |
|    API     +-----------------+
|            |                 |
+------------+                 |
                               |
               +---------------v----------------+
               |                                |
               |                                |
               |                                |         +------------------+
               |                                +-------->+                  |
               |          PROCESSING            |         |     DATABASE     |
               |                                +<--------+                  |
               |                                |         +------------------+
               |                                |
               |                                |
               +----------------+---------------+
                                |
                                v
                         +------+-----+
                         |            |
                         |   DRIVER   |
                         |            |
                         +-----+------+
                               |
                      +--------+-------+
                      V                v
              +--------------+    +--------------+
              |              |    |              |
              |  KUBERNETES  |    |  OPENSTACK   |
              |              |    |              |
              +--------------+    +--------------+
```
