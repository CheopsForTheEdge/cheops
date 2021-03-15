# cheops

Generic prototype to manage replication of resources



# Compiling and running on Goland

Git clone the project then import it from the IDE
Run the project. 
Cf tests for testing


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
