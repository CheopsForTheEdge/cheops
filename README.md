# cheops

Generic prototype to manage replication of resources


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

# Compiling and running on Intellij IDE
Git clone the project then import it from the IDE (project type = maven).

pom.xml contains all project's dependencies. You need to download them first.

For this, open pom.xml, right mouse-click, select maven then reload project.

Build the project then run it. There must be a local etcd running.

Download [etcd binary](https://github.com/etcd-io/etcd/releases) then run it.