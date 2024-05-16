# Consistency

This document explains how cheops manages to ensure a consistent state on all nodes with disconnected nodes.

In Cheops every resource has a fixed set of locations that never evolves for the whole lifetime of the resource. As each site (aka location) can work offline and later synchronize with other sites, we use a specific algorithm to ensure a consistent state everywhere with the help of the user. For the rest of the document we will be talking about a single resource R with a given set of locations S1..Sn. The terms "location", "site" and "node" are used interchangeably.

Remember that sites are working offline-first: all considerations will be given from the point of view of a specific site, and that point of view may very well differ on another site. From a technical point of view objects are synchronized, so the point of view of a given site is the sum of objects that are present locally (because they have been created there, or because they have been synchronized). "object" means either an operation or a reply, which is the result of the execution of an operation on a specific site.

## Consistency classes

Each resource is updated by operations. After being synchronized, operations are run and create a reply; this reply is also synchronized everywhere. Operations can be of any type and are regarded as a black box by Cheops. The only information Cheops needs is its class, or consistency model. This is one of the following four:

- Type 1, or A: Commutative and Idempotent
- Type 2, or B: Commutative and Non-Idempotent
- Type 3, or C: Non-Commutative and Idempotent
- Type 4, or D: Non-Commutative and Non-Idempotent

The user is expected to provide the type along with the command, and Cheops will manage replication, synchronization and such that every location ends up in a consistent state, provided operations do not rely on side-effects.

Type 4 can be analyzed immediately because such an operation cannot be handled by Cheops: if it happens, Cheops bails out and tells the user they can't have a synchronized system.

Type 1 and 2 are similar in that their commutative property will be used to give both types the same treatment. They typically are "mutation" operations: "add 3 to the value".

Type 3 is typically "replacement" operations: "set the value to 7".

## Direct Acyclic Graphs (DAGs)

In Cheops all operations are linked with the parents which are the known set of operations with no children. At insertion time the receiving node will add this list of parents.

At the beginning, it is the empty string. Most of the time there should be only one parent (the operation that happened before) but in some cases, typically when operations are run in parallel, there can be 2 or more, up to the number of sites. We know there is a conflict because there is more than 1 node with no children: these are called leaf nodes. When adding a new operation, if both concurrent operations are known, the "parents" will be all these nodes.

Here's an example of insertion of operations:

```
S1 ----B--C--------
S2 --A----D------F-
S3 -----------E----
```

In this example A comes strictly before B; C and D are inserted at the same time, E comes after C and D, and F comes after E. Supposing that all 3 nodes always synchronize with everyone else, here's the parents for each operation:

- A: [""]
- B: ["A"]
- C: ["B"]
- D: ["B"]
- E: ["C", "D"]
- F: ["E"]

## DAGs solve causality

Whenever a site receives an update of operations (either locally or remotely), we must first ensure that the DAG is complete. In the above example, imagine S1 never received operation E but receives operation F: since S1 can't rebuild the DAG faithfully it can't solve anything and will do nothing (F is still known locally, but not executed; in fact, not even considered). Only when E will be received will S1 actually start to proceed. This behaviour is required for causality (if F was played, the later appearance of E could change things). It is also safe to be re-run, because only when the DAG is complete will the node do anything.

## Converging on a consistent state

As can be seen from the previous example, adding concurrent operations is something that can happen at any time and needs to be dealt with. Here's how we do it.

Type 3 operations are non-commutative and idempotent. This means that the order matters, but once applied they can be applied any number of times; these are "replacement" or "reset" operations. It is easy to intuit that whatever happens before them doesn't really matter: it will be replaced by them. It makes sense then that the last operation of type 3 marks another kind of threshold, because all operations of type 2 after it start from the state of the type 3 operation.

Thanks to this property it is enough to take the last type 3 operation from each site and consider them and the following operation (which will always be type 2). If there are no 3 operations for a site, then it's all 2 and we consider them all from the beginning. We look if the type 3 operations are in conflict, meaning that one is not the ancestor of the other; if there is a conflict we pick one winner based on a deterministic order. Otherwise we take the last one semantically. Once applied, we apply all operations that are causally later, from any site: they will be all type 2 so can be applied in any order, even if they conflict.

We take care to register when an operation has already been run to not re-run it.
