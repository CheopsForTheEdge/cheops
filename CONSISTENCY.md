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

In Cheops each location maintains a log of operations it itself created. This means that when a user sends a new command, it always sends it to a specific site which records it in its log and manages to eventually send it to all involved nodes. By sending, we mean that the log is replicated on all sites; all sites know all lists of all operations from each node, as separate entities:

```
S1 ----A-----B----C---
S2 --D----E-----------
S3 -----------------F
```

In this example, all 3 sites know the exact list of operations that exist on other sites. Recall that this is the view from a given site; for another site, the view might be different because some operations haven't been replicated yet.

## Execution of operations

When an operation is played on a given site, Cheops will create a reply specific to that site and that operation. That reply contains data specific to the execution but not relevant to the consistency model. It will then be replicated to all sites. This means that if there are 3 sites, an operation A will eventually create 3 replies: one from S1, one from S2 and one from S3. Each is created on the respective site and synchronized to the others. Let's call them A1, A2, A3. Eventually one will find the trio on S1, on S2 and on S3.

## Dead or Alive

Operations can be linked with the replies for it:

- Dead: an operation is dead on a site if all replies from all sites have been synchronized to the site, or if on the same site another later operation is dead
- Alive: an operation is alive on a site if not all replies have been synchronized to the site, and if on the same site all operations after it are alive

For example, taking the previous example of A1, A2, A3: if the 3 replies are replicated on S1, then A will be dead on S1. If S2 has only received A1 and A2, then A is still alive on S2 (but eventually will receive A3, so will eventually be dead, or an operation happening after on S2 will be dead).

Another way of looking at these definition is to imagine a monotonically progressing cursor on each site: as the cursor progresses, the operations before are dead, the one after are alive.

## Converging on a consistent state

As the "cursor" advances, some operations become dead and new ones are alive. Alive operations accumulate and need to be executed in a way that converges on all sites. Here's how it's done.

Type 3 operations are non-commutative and idempotent. This means that the order matters, but once applied they can be applied any number of times; these are "replacement" or "reset" operations. It is easy to intuit that whatever happens before them doesn't really matter: it will be replaced by them. It makes sense then that the last operation of type 3 marks another kind of threshold, because all operations of type 2 after it start from the state of the type 3 operation. If we take this set (a type 3, then zero or more type 2 only) and replicate it everywhere the state will be the same.

What we do in Cheops is take the last operation of type 3 of all sites: there will be at most one per site. We compare all of them with a deterministic sorting function (one that doesn't depend on the node, the index of the operation, but only on the operations themselves) and take one of them (always the highest): this is the "winning" set of operations that is thus to be executed everywhere. This ensures a convergent state that makes sense. As usual, once this is done, those operations are marked as dead and the cycle goes on

(As a special case if there are no type 3 at all, it means everything is a type 2: since those are commutative, we apply all that haven't been already applied everywhere)

