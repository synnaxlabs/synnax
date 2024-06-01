# 19 - Cesium Race Conditions

**Feature Name** - Channel Segment Storage Engine <br />
**Status** - Draft <br />
**Start Date** - 2024-03-10 <br />
**Authors** Leo Liu <br />

# 0 - Summary

In this RFC I discuss the nuances of Cesium's concurrency.
More specifically, how it handles race conditions across its different operations.

# 1 - Vocabulary

**Thread** – An execution of program independent of others.
In Go, a thread is implemented as a goroutine, which is more lightweight and compact.

**Concurrency** - Multiple threads (goroutines in the context of Go) running at the same time in parallel.

**Mutex** - Mutual Exclusion, i.e. only one running at once.

**Deadlock** - When two threads wait for each other to finish running or to release some Lock,
therefore become stuck forever.

# 2 - Motivation

Cesium-based atomic clocks are "the most accurate realization of a unit that mankind has yet achieved."
By extension, Cesium, the database named after the element in question, must also be accurate.
This is challenging, however, due to Cesium's concurrent nature.
When multiple subprocesses (goroutines) are to apply operations on the database,
it is vital to handle these potentially conflicting operations in an orderly and predictable manner.

As Cesium looks to become production-ready, this task becomes more important as Cesium must support
Read operations, Write operations, and Delete operations (through a tombstone/garbage-collection system).
All of the operations must coexist without conflicting each other:
for example, we obviously do not want two writers writing to the same  file,
but it's totally fine to read from a file in the first 100 bytes,
while a writer in append-mode is writing to byte 150 and onward.

# 4- Design

The core method through which concurrency is protected is the *Mutex*.

The Mutex allows for operations of threads (going forward,
I will refer to these as goroutines in the specific context as go) to be MUTually EXclusive.
In other words, if one goroutine acquires a mutex (by calling `mu.Lock()`,
all other goroutines trying to acquire it must wait until it is freed, i.e. `mu.Lock()`
will block until the first goroutine releases the Mutex, i.e. `mu.UnLock()`.
In addition, one goroutine may acquire a Read-Lock only (`mu.RLock()`),
which allows other goroutines to acquire Read-Locks, but no goroutine may acquire a write-Lock.

# 4- Analysis

## 4.1 Top-down view of race conditions in Cesium

Protecting the integrity of data in Cesium is crucial.
The three shared resources that we cannot allow concurrent access to are data stored on the file system,
the index stored in memory, and its persisted counterpart on the file system.
We will spend most of our time talking about the first two,
since the persisted index is never read from until the database is closed and reopened.

We will consider which resources each operation needs to access, and in what order.
Consider the following diagram:

<div align="center">
    <img src="img/0019-240529-cesium-race-conditions/Operation-Entities.png" width="90%" />
    <h6>Cesium Entities and Resources</h6>
</div>

We can see that every operation uses the telemetry data in file system,
the index, or one of the two – additionally, if the index is modified,
it must be persisted to the file system as well.
Throughout this RFC, we will discuss which red-coloured operations can occur at the same time,
and how, and which ones cannot.
The general principle we will abide by is that while one goroutine modifies some information,
no one else should read or write from it.
If no goroutines are writing to the resource but all are merely reading from it,
that is acceptable, since all goroutines read the same information.

### 4.1.1 Contentions with reading

It must be noted that the resource management of read is different from the other operations.
Namely, read must be carried out through an iterator, which only reads one chunk of telemetry data at a time.
This means that every time the caller reads a block of telemetry data,
it first gets the information about the location and length of the data in the file system,
then uses that information to read on disk.

With this approach, the caller must understand that if the index was changed between two domains –
say, the iterator was on position 1, and on a call to `Next()`, it moves on to position 2.
However, a writer inserts a new domain between the two, meaning that the iterator thinks it is on position 2,
when in reality it loaded the domain at position 3.

Other than this race condition with the index that the user must beware of,
read does not conflict with any other operations as it does not change the index nor the underlying data.

### 4.1.2 Write-Delete contention

Deletion works by rewriting the pointers where the specified time range's start and end
time stamps are found, and removing all pointers in between.
As deletion does not anything from the underlying telemetry data in the file system,
so we only need to consider the concurrent operations on the index by the writer and the
deleter. We will discuss the two cases: either the writer may write data to the range
requested to be deleted, or it may not.

In the case that the writer may not write to the deleted time range, the mutex on the index
is sufficient to handle the concurrent writes as the writer and deleter are necessarily
writing to different parts of the index, so there will be no conflicts.

The case where the writer may write to the deleted time range, however, will cause index
conflicts: if the write operation happens first, then a new domain will be inserted and
it will be deleted from the deletion operation. However, if the deletion operation happens
first, then there will be a chunk of written data in the time range despite the delete call.
This is not the behaviour we want. Since we cannot know, if called concurrently, whether
the delete happened first or the write, we use the controller to disallow all deletes
to a time range that a writer may write to.

### 4.1.3 Write-GC Contention

Both the writer and the garbage collector modify the underlying data as well as the index.
In addition, since garbage collection is fundamentally a secondary operation (compared
to reads, writes, and deletes), it should not affect those operations – it is unrealistic
to disallow reads and writes while garbage collecting.

Thankfully, by nature, there are major conflicts between a writer and GC: Garbage
Collection is strictly only run on a file that is over size and with no writers currently
writing to it. This means that no writer can write to a file currently being garbage collected.

In the index, since GC only modifies pointers on the file being collected, it will not
change any pointers being written to by the writer, as it is necessarily not on the file,
for reasons mentioned above.

Therefore, these conflicts are taken care of by the file controller and the mutex, and
there are no additional actions needed.

### 4.1.4 GC-Delete Contention

The deleter does not modify the data in the file system, so we will only discuss the index.
Unlike the writer, the deleter may perform operations on the pointers currently being
garbage collected.

The conflict arises when both entities attempt to change the offset of a file: delete
must change this to account for deletion after the start of a domain to only take in
a part of the data; GC must change this to store the new position of the data in the
file system. Thankfully, both changes to the offset are with respect to the existing value:
both operations change the offset by adding (delete) or subtracting (GC) some bytes
to the existing offset. Therefore, it does not matter which one happens first. This way,
GC and delete can work in tandem, so long as the modifications happen one by one and not
at the same time, which is handled by the mutex on the index.

The following diagram summarizes possible conflicts between these entities:

<div align="center">
    <img src="img/0019-240529-cesium-race-conditions/Entity-Interactions.png">
</div>

## 4.2 Bottom-up view of race conditions in Cesium

We will now inspect, from the bottom up, how the various entities interact with the
domain data and the index in memory.

### 4.2.1 Reading

To read a piece of telemetry data on the
