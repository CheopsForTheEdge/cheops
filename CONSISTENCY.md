# Consistency

This document explains how cheops manages to ensure a consistent state on all
nodes with disconnected nodes.

In Cheops every resource has a fixed set of locations that never evolves for
the whole lifetime of the resource. As each site (aka location) can work
offline and later synchronize with other sites, we use a specific algorithm to
ensure a consistent state everywhere with the help of the user. For the rest of
the document we will be talking about a single resource R with a given set of
locations S1..Sn. The terms "location", "site" and "node" are used
interchangeably.

Remember that sites are working offline-first: all considerations will be given
from the point of view of a specific site, and that point of view may very well
differ on another site. From a technical point of view objects are
synchronized, so the point of view of a given site is the sum of objects that
are present locally (because they have been created there, or because they have
been synchronized). "object" means either an operation or a reply, which is the
result of the execution of an operation on a specific site.

## Consistency classes

Each resource is updated by operations. After being synchronized, operations
are run and create a reply; this reply is also synchronized everywhere.
Operations can be of any type and are regarded as a black box by Cheops.  It is
up to the operator to decide which application update is associated to what
operation: in the case of a REST application, the same PUT might be associated
to multiple kinds of operations, depending on the business logic. The only
information Cheops needs is how it interacts with other operations. For each
ordered pair of operations there are 4 relationships possible:

- (1) Take only one operation
- (2) Take both operations in any order
- (3) Take both operations in the order they are given
- (4) Take both operations in the reverse order they are given

These possibilities are expected to be given for every pair of operations that
the application will handle, by the operator, once. If a pair is encountered by
Cheops and has not been configured, it is assumed to be "Take both operations
in any order". The operator will then be tagging each operation that is to be
ran and Cheops will handle synchronization as defined in the next paragraph

## Block

At any given point in time a resource is a list of operations (possibly empty).
When a new operation needs to be added, it is compared to the last one that
already exists:

- if there is none, the new operation is added
- if there is one, we take both in the (existing, new) order and check the
associated relationship
	- if it is 2 or 3, they are "compatible" together: the new operation is
added at the end
	- if it is 1 or 4, they are "incompatible" as-is: the list of existing
operations is discarded and the new operation replaces them

The list of current operations is called block. Every time it changes it is
synchronized to other places, who run the new operations at the end. If there
was a local change, there is a conflict, the resolution of which we will see in
the next section

## Conflict resolution

When there is a conflict, the tool is expected to give Cheops the offending
blocks and a deterministic winner. Because it is deterministic, the same
algorithm can apply on all nodes and will give the same result.

Because of how blocks are built, it is enough to compare only the first
operation of each block: all the operations after can be re-added in the same
order, since they are all "compatible" and could be added at the end before
already. The synchronization layer deterministically gives us a "winning"
version all the time. The first operation of each conflict is compared, with
the one in the winner first, and this gives us the way to resolve the issue:

- if it is the same operation, then keep it and use all operations after
- if the relationship is a 1, then the first operation of the winner is kept,
along with all following operations in the winner and in the conflict
- if the relationship is a 2, 3 or 4, then the operations are ordered as
desired. The operation that comes after is always appended at the end of the
existing operations

This algorithm ensures there is always a single, common list of operations.