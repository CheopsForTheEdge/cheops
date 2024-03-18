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

## Logs

In Cheops each location maintains a log of operations it itself created. This means that when a user sends a new command, it always sends it to a specific site which records it in its log and manages to eventually send it to all involved nodes. Along with the operation, the state as known by the site is recorded to manage concurrent operations. By sending, we mean that the log is replicated on all sites; all sites know all lists of all operations from each node, as separate entities:

```
S1 ----A-----B----C---
S2 --D----E-----------
S3 -----------------F
```

In this example, all 3 sites know the exact list of operations that exist on other sites. The known state of each site for each operation is the index of the last known operation of other sites. For example here, when D was inserted, the state of S1 and S3 was 0. Recall that this is the view from a given site; for another site, the view might be different because some operations haven't been replicated yet.

## Insertion of operations

When an operation is added on a given site, Cheops will push it at the end of its log along with the list of known states.

Let's take the previous example, and assume all 3 nodes are converged: the known state in general will be {S1: 3, S2: 2, S3: 1}. If S2 adds a new operation G, it will record it along with the following: {S1: 3, S3: 1}. If S3 adds 2 new operations H and I at the same site at the same time, it will record it along with the following: {S1: 3, S2: 2}, for the 2 operations

## Converging on a consistent state

As can be seen from the previous example, adding concurrent operations is something that can happen at any time and needs to be dealt with. Here's how we do it.

Whenever a site receives an update of operations (either locally or remotely), all the logs are compared to the last locally known state. In the last example, from the point of view of S3, G is new; from the point of view of S1, G, H and I are new (note: H and I will always arrive in order, but maybe not at the same time; G might also arrive at a later time. Eventually). We take the new operations and all the operations that started after the know states indicated along with them: from S1 point of view, the last local operation (C) indicated that the last know state is {S2: 2, S3: 1}. The new operation are then G, H and I.

Type 3 operations are non-commutative and idempotent. This means that the order matters, but once applied they can be applied any number of times; these are "replacement" or "reset" operations. It is easy to intuit that whatever happens before them doesn't really matter: it will be replaced by them. It makes sense then that the last operation of type 3 marks another kind of threshold, because all operations of type 2 after it start from the state of the type 3 operation.

Thanks to this property it is enough to take the last type 3 operation of the previous set and consider them and the following operation (which will always be type 2). If there are no 3 operations for a site, then it's all 2 and we consider them all. We look if the type 3 operations are in conflict, meaning that one is not the ancestor of the other; if there is a conflict we pick one winner based on a deterministic order. Otherwise we take the last one semantically. Once applied, we apply all operations that are causally later, from any site: they will be all type 2 so can be applied in any order, even if they conflict.
