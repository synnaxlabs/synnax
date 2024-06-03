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

<div style="text-align: center">
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

<div style="text-align: center">
    <img src="img/0019-240529-cesium-race-conditions/Entity-Interactions.png">
</div>

## 4.2 Domain level race conditions

We will now inspect, from the bottom up, how the various entities interact with the
domain data and the index in memory.

### 4.2.1 Writing

A write proceeds in three phases:
1. Acquire a writer (getting a file handle to an existing file or opening a new one)
to an underlying file that still has space for more data.
2. Write the binary telemetry data to the file handle.
3. Commit the writer, i.e. add a pointer in the index storing the time range represented
by the data and the location of the data in the file system (file key, offset, length).

Optionally, Cesium is different from other Database systems in that a writer may be
committed multiple times: a writer may repeat steps 2 and 3, i.e.

4. Write more binary telemetry data to the file handle
5. Commit the writer, i.e. update the pointer that previously described the domain
written by this writer to contain the new domain.

In Synnax version 19, file cutoffs were introduced, which upon a writer writing to a file
that reaches its size limit, automatically ends the domain and begins a new one. We must
also consider the creation of this new pointer when analyzing for race conditions.

#### 4.2.1.1 Acquiring a writer

When acquiring a writer, we must guarantee that we acquire a file handle on a file that
is both exclusive and under the file size limit. We first scan through the open file handles
in the file controller under a `RLock` and upon finding a file within the size limit, we
try to acquire it by comparing-and-swapping its flag (`True` indicates the writer is currently
in use) from `False` to `True`. If successfully acquired, we return that file handle as
our writer and the acquisition of a writer is complete.
Note that although `RLock` does not guarantee exclusivity, the atomic flag
guarantees that only one writer will ever be in control of a file handle.

If none were acquired out of the open file handles, we first calculate the number of file
handles in the file controller under an `RLock`: if it is below the limit, then we may
acquire a new writer – note that there is a possible race condition here (#1): multiple
callers may each try to create a new writer after checking that the number of file handles
is below the limit, pushing the total number of file handles beyond the limit.

In the case where we open a new writer, we lock the mutex of the writer completely with
an exclusive `Lock` and release it when we have a new writer. The acquisition of a writer
is complete.

In the case where we do not open a new writer, i.e. the number of file descriptors exceeds
the limit, we garbage-collect all writers (file handles to oversize files) under an exclusive
lock, and try to acquire a writer again if we garbage-collected any writers. If we did
not, then we wait for a writer to be released (signaled by an input into the channel `release`),
and try to acquire a writer again.

**Race condition #1**: file_controller: acquireWriter's descriptor limit check may not extend
into the following if-statement.

#### 4.2.1.2 Writing data with a writer

`Write` writes binary data into the file handle acquired in the previous step and updates
the `len` and `fileSize` fields of the writer. Note that `Write` may not be called
concurrently with any other Writer methods because it modifies these fields and writes to
the file system.

#### 4.2.1.3 Committing the data in a writer

`Commit` first reads the length of the internal file handle to determine how much data
was written in this domain: note that there is a race condition here (#2): the internal
file handle's length may get changed via another `Commit`, which may assign `internal` a
new writer should there be a file switch, causing `w.internal.len()` to be 0 instead of
the actual length.

Using the `len` field, the writer determines whether a file cutoff is needed. Note that
this step is race-free because `len` may be only updated via `Write`, which is not to be
called concurrently.

Using the length and offset information of the `TrackedWriterCloser`, the index is
inserted with the new pointer under an exclusive lock, therefore race-free. After the
insertion, the writer may switch files, changing the `fileKey`, `internal`, `fileSize`
fields. This is the same race condition as mentioned before (#2), where other `Commit`
calls read from this modified field.

**Race condition #2**: domain/writer: `Commit` modifies fields that other `Commit` calls
may use.

**Recommended fix**: Disallow concurrent calls to `Commit` – in addition to guaranteeing
exclusivity, it also does not make sense to make parallel calls to `Commit` if they all
have the same underlying data as `Write` may not be called concurrently.

### 4.2.2 Reading

In Synnax, all readings are handled by the entity _Iterator_, which allows reading of
telemetry data in domains and prevents reading of potentially massive data ranges entirely
into memory. For this reason, there does not need to be any logical order in which these
domains are stored on the file system – that is tracked by the index – so the iterator's
job is to determine where to read and create a File System reader that can only read that
section of the file.

At the domain level, reading goes through three phases:
1. Using commands such as `Seek`, `Next`, or `Prev`, find the position of the domain of
interest that contains the time range we wish to read.
2. Load the pointer at the found position into the iterator as its `value` field.
3. Create a section reader based on information in the stored pointer: the offset for
this reader is set to 0 where the domain starts in the file system, and EOF where the
length of the domain is reached.
4. Use the section reader as an `ReaderAtCloser` to read binary data.

Note that there are various race conditions here! These race conditions are a
combination of three categories: change to the pointer itself, change to the index, and
change to the underlying data.

At step 2, changes made to the index prevent `i.position` from pointing to the right
pointer; at step 3, changes made to the pointer prevent creating the right reader; at
step 4, changes made to file prevent reading the correct data. Here is a diagram that
summarizes these races:

<div style="text-align: center">
<img width="90%" src="img/0019-240529-cesium-race-conditions/Read-Races.png">
</div>

#### 4.2.2.1 Race conditions involving the index
If the index's content was changed between finding the position of the pointer of interest
and reading it, the iterator will not read the correct pointer. For example, assume the
seeked position of interest is 3. These changes may lead to reading the incorrect pointer:
- `Write` inserting a pointer before the position of interest: the pointer of interest
is found at position 6 instead of 5.
- `Delete` deleting pointer(s) before the position of interest: the pointer of interest
is found at position 4 instead of 5.
- `Delete` deletes the pointer at the position of interest: the pointer of interest is
no longer in the index, instead we would be reading what was at position 6.

Notice that changes to the index after the position of interest does not invalidate the
position.

Also note that not only do these changes invalidate loading the pointer into the iterator,
they also invalidate future iterator movements – the position of the iterator is no
longer consistent with the value.

#### 4.2.2.2 Race conditions involving the pointer
Note that for creation of the reader, the offset and length information are not based on
the pointer stored in memory, but the pointer loaded into the iterator. Therefore, if the
pointer of interest was modified, the reader would not be updated with these new changes.

This is not necessarily a problem – the iterator stores a pointer that was correct
_at some point in time_ – but it may be inconvenient: for example, consider an
open reader on domain that gets keeps getting updated with consecutive `Write`s: the reader
would maintain the original domain's length and end time stamp until it is reloaded.

Operations that could cause this discrepancy between the iterator-stored pointer and the
pointer in index are:
- Amend-`Write`s
- `Delete`s
- Garbage-Collection.

#### 4.2.2.3 Race conditions involving the file

If a file is garbage collected after a reader has been created on it, the offset and length
of the pointer of interest are no longer coherent with the location of telemetry in the
file. This causes reading of wrong data. The change in the file could only happen from
one operation:
- Garbage-Collection.

#### 4.2.3 Deletion

At the domain level, deletion is executed using two locks: one lock to prevent other
deletions from occurring at the same time, and another lock to lock the mutex while
we modify it. The specific steps to deletion are as follows:
1. Find the domains containing the starting and ending time stamps.
2. Calculate the offsets (number of samples) from the start time stamps of those domains
to the desired time stamps.
3. Search the domains where start and end time stamps found again.
4. Remove pointers in between and update the domains at the two ends with the new offsets.

Note that step 3 is necessary because between calculating the offsets, the position of the
start and end domains may have changed – however, what must not have changed are the
content of those domains, as we will never have an open writer during deletion and the
delete lock prevents other deletions from occurring concurrently – therefore we know the
start and end offsets remain correct.


