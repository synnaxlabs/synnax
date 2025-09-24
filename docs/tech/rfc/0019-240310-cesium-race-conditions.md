# 19 - Cesium Race Conditions

- **Feature Name** - Channel Segment Storage Engine
- **Status** - Complete
- **Start Date** - 2024-07-05
- **Authors** - Leo Liu

## 0 - Summary

In this RFC I discuss the nuances of Cesium's concurrency. More specifically, how it
handles race conditions across its different operations.

## 1 - Vocabulary

**Sample**: A sample is a value in a strongly-typed channel recorded at a specific
moment in time. We will be using the term "sample" instead of the broader term "data
point" in the context of Cesium.

**Channel**: A channel is a logical collection of samples that are time-ordered and
share the same data type. Typically, a channel is used to store a time-series collection
of samples from the same source, e.g. a sensor.

**Index**: In a channel, we may find many samples, but the samples don't make sense
unless we know what time stamp each sample corresponds to (is recorded at). An index
_indexes_ a channel by associating a time stamp to each sample. It does not make sense
to query the data at 00:05 out of [10, 30, 23, 90], but it does once we know that those
samples correspond to [00:03, 00:04, 00:05, 00:10]. It is necessary that each channel
has an index, and note that the index itself must be stored in a channel – all samples
are time-ordered and share the data type of timestamps.

**Telemetry data**: At its core, all of Cesium's samples are stored as binary data on
the file system. Samples of the same [data type](0007-220823-data-type.md) occupy the
same number of bytes: for example, to read data from a file containing int64 data, we
read every 8 bits and marshall each one into decimal form to recover the samples.

**Domain**: A domain is a time range with many samples stored contiguously in the same
file. In Cesium, **domains must not overlap each other**: the same moment in time cannot
have two different data points. These properties render the domain a powerful concept
because it allows us to make statements about an entire group of samples: if we know
that the time range 10:00 - 12:00 is stored in file 1 from offset 10 with length 10, and
we want to find the sample at 11:00, we know exactly where to look for it – this
prevents us from searching the entire database!

**Pointer**: A pointer is the implementation of a domain: a pointer is a `struct` that
stores the time range of the domain and where to find data for the domain in the file
system (file key, offset, length).

**Domain index**: The domain index is a chronologically-sorted slice of pointers stored
in memory. The pointers are sorted by their time range and no domain may overlap another
(as discussed in domain). The domain index allows us to find the domain that contains a
given time stamp (and does this fast via binary search) or the lack thereof.

**Index persist**: Index persist is a permanent copy of the domain index to the file
system. Suppose the database closes then reopens and content in the memory is erased –
we can no longer find data as we no longer have index in our memory. Index persist is an
exact replica of the domain index on the file system: if the index is lost, we can
reload it from the persisted version. Each pointer is encoded into 26 bytes, and the
entire slice of pointers is stored in the `index.domain` file. Index persist must be
closely up-to-date with the domain index – however, there is a tradeoff between persist
frequency and performance.

**domainDB**: The `domainDB` is the lowest layer of abstraction in Cesium: it is
responsible for storing and retrieving telemetry data from files. It does so by managing
the domain index, index persist, and the file handler which grants direct access to
files in the file system.

<div align="center">
    <img src="img/0019-240529-cesium-race-conditions/Index-Concepts.png" width="90%" />
    <h6>Domain-level concepts</h6>
</div>

**unaryDB**: The `unaryDB` is the layer above the Domain DB in Cesium. While the latter
is responsible for storing concrete data, the Unary DB is responsible for associating
the time-series data with the correct timestamps via an _index_.

**cesiumDB**: The `cesiumDB` is the highest level of abstraction in the storage engine:
it comprises multiple `unaryDB`s to represent multiple channels.

## 2 - Motivation

Concurrency is a double-edged sword: while it could make Cesium blazing-fast, it
introduces additional complexity through parallelism. As Cesium looks to become
production-ready, it is crucial that Cesium must support read, write, and delete
(through a tombstone and garbage collection system) operations at high speeds without
them conflicting each other. One can imagine the possible consequences for unhandled
race conditions in extreme scenarios such as launch control systems: it would not be
desirable.

## 3- Analysis

### 3.0 Primer on the Structure of Cesium

As a primer for analyzing race conditions in Cesium, we will first analyze how Cesium is
structured and how it stores data. While all the actual telemetry data is stored on the
file system, the memory assists in storing information as to where on the file system a
desired chunk of data may be found.

#### 3.0.1 Cesium's file storage structure

All of Cesium's data is stored in the root directory specified by the user when Cesium
is booted. Each channel stores its data in a subdirectory of the same name as the
channel key: this subdirectory is the file system of that channel. Within this file
system, a `meta.json` file is used to store the metadata of this channel, e.g. key,
rate, etc. This file is not relevant for our purposes of analyzing race conditions.

The channel's file system also includes the `index.domain` file used for domain index
persist. All other files in the file system are used to store binary telemetry data –
they are numerically labeled and more files get created as existing files get filled up
– i.e. `1.domain`, `2.domain`, and so on.

<div align="center">
<img src="img/0019-240529-cesium-race-conditions/File-System-Structure.png" width="90%">
<h6>File System structure</h6>
</div>

#### 3.0.2 Cesium's database structure

At the highest level of abstraction, Cesium provides one unified interface to read,
write, delete, and stream data from multiple channels. A `cesiumDB` comprises many
channels, each one of which corresponding to a `virtualDB` or an `unaryDB`. A
`virtualDB` contains no data, so it will not be a topic of our discussion.

A `unaryDB` is a second level of abstraction in the database structure: it is composed
of a `domainDB` an index, and the controller: the index is responsible for querying the
timestamps corresponding to telemetry data, the controller is responsible for
controlling writes to the database between different writers, and the `domainDB` is
responsible for storing the actual telemetry data.

A `domainDB` is the lowest level of abstraction: it provides means to write and read
data from the file system, and maintains the domain index.

<div align="center">
<img src="img/0019-240529-cesium-race-conditions/Cesium-Structure.png" width="90%">
<h6>Database structure</h6>
</div>

Each layer has their own entities and methods of reads, writes, deletions, and garbage
collection. All interactions are passed down to the `domainDB` to interact with the
domain index and the file system.

### 3.1 High-Level View of Race Conditions in Cesium

#### 3.1.0 Types of possible race conditions

There are three types of race conditions that we are most concerned with. The first is
concurrent modification of the same resource: this must be strictly forbidden as it is
almost always erroneous. Consider the following code that intends to add 3 and subtract
1 from 1 – as we know, this should result in 3.

```go
var sharedInt = 1

// goroutine 1
func add(){
    sharedInt += 3
}

// goroutine 2
func sub(){
    sharedInt -= 1
}

go add()
go sub()
```

Both goroutines try to access the same variable, `sharedInt`, leading to potential
errors: for example, if `add` and `sub` are run at the same time, they may each read
`x`'s value as 1 and effectuate their operation on 1, leading the end result to become 0
or 4 as opposed to the desired value of 3. The above type of race conditions can be
caught thanks to golang's built-in race detector.

The second type, however, is more subtle. Consider the following code that introduces a
mutex to fix the race condition:

```go
var (
    sharedInt = 1
    mu sync.Mutex
)

func isPositive() bool{
    mu.Lock()
    defer mu.Unlock()
    return sharedInt > 0
}

// goroutine 1
func addOnlyToPositive(){
    if isPositive(){
        mu.Lock()
        sharedInt += 3
        mu.Unlock()
    }
}

// goroutine 2
func sub(){
    mu.Lock()
    sharedInt -= 1
    mu.Unlock()
}

go addOnlyToPositive()
go sub()
```

The code eliminates race conditions of the first type: all accesses to the variable
`sharedInt` are in series and not parallel. However, another problem arises: since we
unlock the mutex during the execution of `checkCondition()`, by the time we lock it
again to add to it – the condition may no longer be satisfied anymore, causing undesired
behavior. This type of race conditions generally involve concurrent – but not parallel –
modification to a field, causing following statements to be incorrect. Race conditions
of this type are very subtle and hard to catch; once caught, they usually require a
considerable amount of refactoring.

The last type of race conditions are deadlocks: this is when two goroutines try to
acquire the same mutex lock, like in the following scenario:

```go
var (
    mu1, mu2 sync.Mutex
)

func fun1(){
    mu1.Lock()
    mu2.Lock()
    ...
    mu2.Unlock()
    mu1.Unlock()
}

func fun2(){
    mu2.Lock()
    mu1.Lock()
    ...
    mu1.Unlock()
    mu2.Unlock()
}

go fun1()
go fun2()
```

In this scenario, `fun1` acquires a lock on `mu1` and `fun2` acquires a lock on `mu2`.
As `fun1` tries to acquire a lock on `mu2`, it can't as `fun2` holds it – the same goes
for `mu1` – the execution in the thread of `fun2` is also blocked. Therefore, both
threads are blocked and the program never halts, resulting in a deadlock.

In Cesium, most race conditions are of the second type, as the other two types results
are easily debuggable.

#### 3.1.1 Overview of Cesium's operations

The three shared resources that may be accessed concurrently are data stored on the file
system, the index stored in memory, and its persisted counterpart on the file system. We
will mainly discuss the first two entities and omit the persisted index, since it is
never read from until the database is closed and reopened.

Each one of the read, write, delete, and garbage collection operations is a combination
of accesses to the index and telemetry data. In the following diagram, we offer an
overview of which resources are involved with each operation. We will briefly discuss
how each operation is carried out in this section, and analyze the race conditions with
each in detail in following sections.

<div align="center">
    <img src="img/0019-240529-cesium-race-conditions/Operation-Entities.png" width="90%" />
    <h6>Cesium entities and resources</h6>
</div>

##### 3.1.1.1 Cesium reads

Read operations are carried out through an iterator, which only reads one chunk of
telemetry data at a time – this is desirable since it reduces the memory overhead given
rarely would one need data on the entire time range in the database all at once. Every
time the caller wishes to read telemetry data, the iterator first locates the domain
where the data starts via seek in the domain index, uses the information stored in the
domain's pointer to create a reader, and reads the data from the file system with the
reader; the iterator moves on to every next domain until the entire time range of
interest is read.

##### 3.1.1.2 Cesium writes

Write operations are performed through a writer. A write occurs in 3 phases:

1. Opening a writer
2. Writing data to the writer
3. Committing the writer

Upon commit, a new pointer is inserted into the domain index to record the location of
the newly written data, enabling it for reads and other operations. Note that a writer
may be committed multiple times (see discussion in [3.2.1](#421-writing)).

##### 3.1.1.3 Cesium deletions

A deletion in Cesium is performed by rewriting the pointers where the specified time
range's start and end time stamps are found, and removing all pointers in between.
Deletion itself executes no file system operations to delete samples. Rather, it
accomplishes the 'deletion' by directly changing domains' lengths and offsets in the
domain index. The data in the file system not in any domain is only removed in the
garbage collection stage.

##### 3.1.1.4 Cesium garbage collection

A garbage collection operation is performed by first choosing a file that does not have
any open writers and is oversize. Then, a replica of that file is created without the
deleted data. Finally, the pointers in the domain index are modified to record the
updated offsets and the original file is swapped for the replica.

#### 3.1.2 Contentions with reading

Reading does not modify data in the domain index nor in the file system. Therefore, it
will not contend with other operations. However, other operations that modify the domain
index or the file system will affect correct reads – see discussion in
[3.2.2](#422-reading).

#### 3.1.3 Write-Delete contention

The deleter does not change data on the file system, so we only need to consider the
concurrent operations on domain the index by the writer and the deleter. We will discuss
the two cases: either the writer may write data to the range requested to be deleted, or
it may not.

In the case that the writer does not write to the deleted time range, the mutex on the
domain index is sufficient to handle the concurrent writes as the writer and deleter are
necessarily writing to different parts of the domain index, so there will be no
conflicts.

The case where the writer may write to the deleted time range, however, will cause
domain index conflicts: if the write operation happens first, then a new domain will be
inserted and then deleted from the deletion operation. However, if the deletion
operation happens first, then there will be a chunk of written data in the time range
despite the delete call. This is not the behavior we want. So we use the controller to
disallow all deletions to a time range that a writer may write to.

#### 3.1.4 Write-Garbage collection Contention

Both the writer and the garbage collector modify the underlying data as well as the
index. In addition, since garbage collection is fundamentally a secondary operation
(compared to reads, writes, and deletions), it should not affect those operations – it
is unrealistic to disallow reads and writes while garbage collecting.

There are no major conflicts between a writer and garbage collection: garbage collection
is strictly only run on a file that is oversize and has no writers currently writing to
it. This means that no writer can write to a file currently being garbage collected.

In the domain index, since garbage collection only modifies pointers on the file being
collected, it will not change any pointers being written to by the writer, as it is
necessarily not on the file, for reasons mentioned above.

These conflicts are taken care of by the file controller and the mutex, and there are no
additional actions needed.

#### 3.1.5 Delete-Garbage collection contention

The deleter does not modify the data in the file system, so we will only discuss the
contentions in the domain index. Unlike the writer, the deleter may perform operations
on the pointers currently being garbage collected.

The conflict arises when both entities attempt to change the offset of a file: delete
must change the offset to account for deletion after the start of a domain to only take
in a part of the data. Garbage collection must also change the offset to store the new
position of the data in the file system. Both changes to the offset are with respect to
the existing value: both operations change the offset by adding (delete) or subtracting
(garbage collection) some bytes to the existing offset. Therefore, it does not matter
which one happens first. This way, garbage collection and delete can work in tandem, so
long as the modifications happen one by one and not at the same time, which is handled
by the mutex on the domain index.

#### 3.1.6 Write-Write contention

In Cesium, write-write contentions are handled via the controller: for a given time
range in a given channel, the controller will only authorize one writer to write and
commit data at a time.

However, the underlying race condition here is that the check on whether a writer has
authority only happens before data is written to the file system – therefore, it is a
possibility that while writer A is writing a large series to the database, writer B,
with a higher authority, gets a call to `Write` and starts writing to the database –
note that since they are two different writers, they are writing to different files in
the domain; upon commit, only one commit can happen at once, and if the two writers
write on the same time ranges, only one commit succeeds. Therefore, race condition is
properly handled by the controller.

#### 3.1.7 Garbage collection-Garbage collection contention

If two garbage collection subprocesses run at the same time, we risk a serious data
race: the underlying data in the file system may be modified by two entities at the same
time, and the offset will be in a bad state.

#### 3.1.8 Delete-Delete contention

There are two cases for delete-delete contention: either the two time ranges contend
over the same pointer(s), or they are disjoint: the case where they are disjoint is
simple: we can simply use mutex locks to turn the operations into serial operations to
the domain index and there would be no conflicts (see the detailed deletion section
3.2.3).

If the two deletions contend over the same pointers, there are two more cases: the two
time ranges could overlap each other, in which case the controller in the `unaryDB`
would return an error and disallow the deletion, or they don't overlap each other, in
which case the operation is turned into a serial operation of the domain index (see
section [3.2.3](#423-deletion)).

The following diagram summarizes possible conflicts between these entities:

<div align="center">
    <img src="img/0019-240529-cesium-race-conditions/Entity-Interactions.png" width="90%">
    <h6>Cesium entities' interaction</h6>
</div>

### 3.2 domainDB Race Conditions

We will now inspect, from the bottom up, how the various entities interact with the
domain data and the domain index in memory. The `domainDB` is responsible for managing
the underlying file system and the domain index. A `domainDB` is index-agnostic; it
knows nothing about the time stamps that its samples correspond to. Therefore, data
written to and read from `domainDB` are simple byte arrays with no information about
timing.

#### 3.2.1 Writing

We will simplify the writer to have the following properties that we will need when
analyzing race conditions:

```go
type Writer struct {
    // idx is the underlying domain index for the database that stores locations of domains in FS.
    idx *index
    // fileKey represents the key of the file written to by the writer.
    fileKey uint16
    // fc is the file controller for the writer's FS.
    fc *fileController
    // fileSize is the writer's file's size
    fileSize telem.Size
    // len is the number of bytes written by all internal writers of the domain writer.
    len int64
    // internal is a TrackedWriteCloser used to write telemetry to FS.
    internal xio.TrackedWriteCloser
}
```

A write proceeds in three phases:

1. Acquire a writer (getting a file handle to an existing file or opening a new one) to
   an underlying file that still has space for more data.
2. Write the binary telemetry data to the file handle.
3. Commit the writer, i.e. add a pointer in the domain index storing the time range
   represented by the data and the location of the data in the file system (file key,
   offset, length).

Cesium is different from other database systems in that a writer may be committed
multiple times. A writer may repeat steps 2 and 3, i.e.

4. Write more binary telemetry data to the file handle
5. Commit the writer, i.e. update the pointer that previously described the domain
   written by this writer to contain the new domain.

In v0.19.0, file cutoffs were introduced, which upon a writer writing to a file that
reaches its size limit, automatically ends the domain and begins a new one. We must also
consider the creation of this new pointer when analyzing for race conditions.

##### 3.2.1.1 Acquiring a writer

When acquiring a writer, we must guarantee that we acquire a file handle on a file that
is both exclusive and under the file size limit.

In the file controller, file handles are wrapped with an atomic flag to indicate whether
it is in use. We first scan through these file handles under a `RLock` in the file
controller and upon finding a file within the size limit, we try to acquire it by
comparing-and-swapping its atomic flag from `False` to `True`, indicating the file is in
use. If successfully acquired, we return that file handle as our writer and the
acquisition of a writer is complete. Note that although `RLock` does not guarantee
exclusivity, the atomic flag guarantees that only one writer will ever be in control of
a file handle.

If none were acquired out of the open file handles, we first calculate the number of
file handles in the file controller under an `RLock`: if it is below the limit, then we
may acquire a new writer – note that there is a possible race condition here: multiple
callers may each try to create a new writer after checking that the number of file
handles is below the limit, pushing the total number of file handles beyond the limit.

In the case where we open a new writer, we lock the mutex of the writer completely with
an exclusive `Lock` and release it when we have a new writer. The acquisition of a
writer is complete.

In the case where we do not open a new writer, i.e. the number of file descriptors
exceeds the limit, we garbage collect all writers (file handles to oversize files) under
an exclusive lock, and try to acquire a writer again if we garbage collected any
writers. If we did not, then we wait for a writer to be released (signaled by an input
into the channel `release`), and try to acquire a writer again.

##### 3.2.1.2 Writing data with a writer

`Write` writes binary data into the file handle acquired in the previous step and
updates the `len` and `fileSize` fields of the writer. Note that `Write` may not be
called concurrently with any other Writer methods because it modifies these fields and
writes to the file system.

##### 3.2.1.3 Committing the data in a writer

`Commit` first reads the length of the internal file handle to determine how much data
was written in this domain: note that there is a race condition here: the internal file
handle's length may get changed via another `Commit`, which may assign `internal` a new
writer should there be a file switch, causing `w.internal.len()` to be 0 instead of the
actual length.

Using the `len` field, the writer determines whether a file cutoff is needed. Note that
this step is race-free because `len` may be only updated via `Write`, which is not to be
called concurrently.

Using the length and offset information of the `TrackedWriterCloser`, a pointer is
inserted into the domain index under an exclusive lock, therefore race-free. After the
insertion, the writer may switch files, changing the `fileKey`, `internal`, `fileSize`
fields.

Note that these race conditions resulting from concurrent calls to `Commit` should never
happen, as the method is documented to not be called concurrently and in Cesium's
implementation, no writer may concurrently call two writer methods.

#### 3.2.2 Reading

We will simplify the iterator have the following properties that we will need when
analyzing race conditions:

```go
type Iterator struct {
    // position stores the current position of the iterator in the idx.
    position int
    // idx is the index that the iterator is iterating over.
    idx *index
    // value stores the pointer of the domain the iterator is currently on
    value pointer
    // readerFactory gets a new reader for the given domain pointer.
    readerFactory func(ctx context.Context, ptr pointer) (*Reader, error)
}
```

In Synnax, all readings are handled by the iterator, which allows reading of telemetry
data in domains and prevents reading of potentially massive data ranges entirely into
memory. For this reason, there does not need to be any logical order in which these
domains are stored on the file system – that is tracked by the domain index – so the
iterator's purpose is to determine where to read and create a file system reader that
can only read that section of the file.

In a `domainDB`, reading goes through three phases:

1. Using commands such as `Seek`, `Next`, or `Prev`, find the position of the domain of
   interest that contains the time range we wish to read.
2. Load the pointer at the found position into the iterator as its `value` field.
3. Create a section reader based on information in the stored pointer: the offset for
   this reader is set to 0 where the domain starts in the file system, and EOF where the
   length of the domain is reached.
4. Use the section reader as an `ReaderAtCloser` to read binary data.

Note that there are various race conditions here! These race conditions are a
combination of three categories: change to the pointer itself, change to the domain
index, and change to the underlying data.

At step 2, changes made to the domain index prevent `i.position` from pointing to the
right pointer; at step 3, changes made to the pointer prevent creating the right reader;
at step 4, changes made to file prevent reading the correct data. Here is a diagram that
summarizes these races:

<div align="center">
<img width="90%" src="img/0019-240529-cesium-race-conditions/Read-Races.png">
    <h6>Life cycle of a domain iterator</h6>
</div>

##### 3.2.2.1 Race conditions involving the domain index

If the domain index's content was changed between finding the position of the pointer of
interest and reading it, the iterator will not read the correct pointer. For example,
assume the seeked position of interest is 3. These changes may lead to reading the
incorrect pointer:

- `Write` inserting a pointer before the position of interest: the pointer of interest
  is found at position 6 instead of 5.
- `Delete` deleting pointer(s) before the position of interest: the pointer of interest
  is found at position 4 instead of 5.
- `Delete` deletes the pointer at the position of interest: the pointer of interest is
  no longer in the domain index, instead we would be reading what was at position 6.

Notice that changes to the domain index after the position of interest does not
invalidate the position.

Also note that not only do these changes invalidate loading the pointer into the
iterator, they also invalidate future iterator movements – the position of the iterator
is no longer consistent with the value.

##### 3.2.2.2 Race conditions involving the pointer

Note that for creation of the reader, the offset and length information are not based on
the pointer stored in memory, but the pointer loaded into the iterator. If the pointer
of interest was modified, the created read does not read the correct chunk of data.

This is not necessarily a problem – as documented, the iterator does not iterate over
pointers in a snapshot of the `domainDB`. but rather stores a pointer that was correct
_at some point in time_. However, this behavior may be inconvenient: for example,
consider an open reader on domain that gets keeps getting updated with consecutive
`Write`s: the reader would maintain the original domain's length and end time stamp
until it is reloaded.

Operations that could cause this discrepancy between the iterator-stored pointer and the
pointer in domain index are:

- Amend-`Write`s
- `Delete`s
- Garbage collection

##### 3.2.2.3 Race conditions involving the file

If a file is garbage collected after a reader has been created on it, the offset and
length of the pointer of interest are no longer coherent with the location of telemetry
in the file. This causes reading of wrong data. The change in the file could only happen
from garbage collection.

#### 3.2.3 Deletion

In a `domainDB`, deletion occurs in four steps:

1. Find the domains containing the starting and ending time stamps.
2. Calculate the offsets (number of samples) from the start time stamps of those domains
   to the desired time stamps.
3. Search the domains where start and end time stamps found again.
4. Remove pointers in between and update the domains at the two ends with the new
   offsets.

<div align="center">
<img width="90%" src="img/0019-240529-cesium-race-conditions/Delete-Races.png">
    <h6>Life cycle of a deletion</h6>
</div>

Deletion employs 3 locks to ensure the integrity of the operation. We first note the
possible race condition between steps 2 and 3 – the offset on the two pointers may be
incorrect if they are modified by another deletion. To address this, we introduce the
delete lock: since there cannot be any `Write`s on the pointers affected in a deletion,
if we also disallow delete for those pointers, we effectively shut down all operations
that could change that section of the domain index. Therefore, we can guarantee that the
offsets stay correct despite the position of the pointers changing (consider, for
example, inserting a domain before the deletion range).

Also note that since the calculation in step 2 is based off the time stamp rather than
the pointer positions, the possible changes to the pointers between steps 1 and 2 does
not affect the offset calculation.

#### 3.2.4 Garbage collection

Garbage collection in `domainDB` rewrites the underlying telemetry data and modifies the
domain index to contain the new offsets. This is done in 8 steps:

1. Garbage Collect writers (i.e. close and discard file handles on oversize files).
2. Iterate over files in the domain database.
3. For a given file, check that it is oversize and has no open writers on it.
4. Read and store a copy of all pointers for domains on that file.
5. Transcribe pointers
   1. Open a reader on the file.
   2. Create a copy file in write mode
   3. Using the pointers, read in the domains and re-write them in the new file, while
      keeping track of the change in offset.
6. Rename the new file to the old file, delete the old file.
7. Change the pointer offsets in the domain index.
8. Repeat steps 3-7 for each file.

<div align="center">
<img width="90%" src="img/0019-240529-cesium-race-conditions/GC-Races.png">
    <h6>Life cycle of a garbage collection</h6>
</div>

Note that a read lock is applied on the domain index for step 4, and an exclusive lock
is applied on the domain index for steps 6 and 7, disabling read and write operations.

There could be a race condition between steps 4 and 5: the pointers may change between
when we first read them to when we transcribe them, however, we can reason that these
race conditions can be handled properly:

- We know that these changes to domain index may only be deletions, as there cannot be
  any additional writes to pointers on a closed file without open writers.
- Garbage collection will only modify the offsets of these pointers, so the only field
  of contention within these pointers in garbage collection and deletion is the `offset`
  field.
- We observe that the change in the offset is additive: if a pointer's offset was
  increased by 3 to exclude the first 3 samples, then the transcribed version of the
  pointer in the new file must also have its offset increased by 3: the order of these
  two changes to offset does not matter.

We have shown that we can handle race conditions on the domain index properly. However,
there may also be a race condition on the file: at the time we transcribe the pointers,
the location of domains in the file may no longer be where they are in the domain index:
this could only be due to another garbage collection process as, as mentioned before,
there cannot be `Write` operations on the pointers affected by garbage collection.

To this end, race condition occurs when there are two garbage collection processes
running concurrently. A fix could be to ensure that this never happens.

### 3.3 unaryDB Race Conditions

In a `unaryDB`, the previously index-agnostic `domainDB` is paired with an index,
allowing us to match a time stamp to each sample. Therefore, contrary to a `domainDB`,
we see that data written to and read from `unaryDB` are series (unmarshalled byte arrays
with a time range), since we can unmarshal the data using the data type proper to the
channel, and we can assign a time range because of the existence of an index.

At this level, not only must we consider race conditions that come up from the
`domainDB`, we must also consider the interaction between a domain database and its
index, which is another domain database.

#### 3.3.1 Controller

To analyze the race conditions involved with writing, we will first consider the
implications of race conditions in the controller.

The controller is responsible for giving the writing entity – a domain writer, for
example – to the writer which has authority to write when there are multiple writers
willing to write to the same time region. Time regions must not overlap each other, and
there must only be one gate controlling a time region at a time.

##### 3.3.1.1 Region operations

A region's main operations are to manage its gates, i.e. the opening, releasing, and
updating of gates. Each one of these operations are completed within an exclusive lock
of the region – i.e. whenever a property of region is updated, we must ensure that we
have exclusive access.

There is a race condition in a peculiar case where the last gate in a region is
released. Upon releasing the gate, the region locks its mutex in order to remove the
gate. After that operation returns, if the gate just removed was the last gate in the
region, then a call to `Controller` is made to remove that region from the controller
entirely.

The race condition lies in that a new region may be inserted into the region after the
gate was released but before the region was removed, causing an inaccessible gate in a
region no longer in the controller. A fix would be to hold the region's mutex lock until
it is completely removed.

##### 3.3.1.2 Controller operations

A controller's main operations are to manage its regions: it may register regions,
remove regions, and insert a gate into a region. All of these operations are performed
under an exclusive lock of the controller. If insertion into a region is involved, an
exclusive lock on the region is also applied.

#### 3.3.2 Writing

Writing to a `unaryDB` is simply an extension from `domainDB`, except with more
complications with respect to the `unaryDB` structures controller and index. Each write
is consisted of 4 steps:

1. Create a writer and store it in the controller region.
2. Acquire the writer from the controller if the writer has authority.
3. Write telemetry data.
4. Commit telemetry data.

Since writer methods may not be called concurrently and steps 2 - 3 do not introduce any
new behaviors other than their domain counterparts, there are no associated race
conditions. For step 4 – committing the telemetry data – in the current implementation
of Cesium, `CommitWithEnd` will always be called with an end time stamp, so we will not
look into the race conditions involved with `stamp` operations when there is no end time
stamp.

#### 3.3.3 Reading

Just like in `domainDB`, reading in Unary is also handled by the iterator, however, the
Unary iterator directly reads data into a frame as opposed to merely providing a means
to read data (like the domain iterator). In addition, moving the iterator in a `unaryDB`
via `Next` requires an additional argument – the time span to move the iterator by: i.e.
the iterator does not move to the next domain, but rather moves to contain the next
`span` time span.

The two main types of operations of a unary iterator are `Seek`s – i.e. moving the
iterator to a specific position of the domain index and `Next`/`Prev` – i.e. reading
data. The `Seek` operations are simple forwarding calls to `Seek` calls in `domainDB`,
so the race conditions mentioned in 4.2.2.1 apply here as well.

`Next` is the means of reading data of the iterator – and note that `Prev` is exactly
congruent, but in the opposite direction. We will analyze the race conditions that might
occur during a `Next` operation. The `Next` operation is simple:

While the iterator's frame does not contain all of its view:

1. In the index, find where in the current domain the desired time range is located.
   This location is represented as an offset and a length. (Note that only for the first
   and last domain, we don't read the entire domain – i.e. the distance calculation is
   only done at most twice.)
2. Based on the offset and length, read the binary data into a series and insert it into
   the iterator's frame in chronological order (i.e. append if we are reading `Next` and
   prepend if we are reading `Prev`)
3. Move the internal iterator forward to the next domain.

<div align="center">
    <img width="90%" src="img/0019-240529-cesium-race-conditions/Unary-Read-Races.png">
    <h6>Unary iterator's <code>Next</code> operation</h6>
</div>

In step 1, this requires opening an iterator in the index DB and finding the domain
containing the desired time range, creating a reader, and reading from it. Therefore,
all the race conditions with the domain iterator also apply here – i.e. the pointer
loaded into the domain iterator, whose time range the unary iterator relies on to
calculate the domain slices, may be incorrect or outdated.

In step 2, this requires that the underlying file's data in the specified location by
the offset and length found in step one has not changed – this is a possible race
condition with garbage collection.

In step 3, this involves moving the domain iterator, so all the race condition in a
`domainDB` also apply here.

#### 3.3.4 Deleting

Deletion in a `unaryDB` is a direct forward of deletion from `domainDB`. The only added
logic is the opening of an absolute gate on the time range being deleted – this will
invalidate all deletions on ranges where a writer may write as well as all deletions
whose time range intersect with another deletion's time range. Once the gate is created,
since it has absolute authority, it will remain in control for the entirety of the
delete duration.

#### 3.3.5 Garbage collection

Garbage collection in a `unaryDB` is a direct forward of garbage collection from
`domainDB`.

### 3.4 cesiumDB Race Conditions

In a `cesiumDB`, multiple `unaryDB`s are stored to represent multiple channels.
Therefore, reads and writes at this level culminate in **frames** (series from different
channels).

In a `cesiumDB`, each entity may control multiple channels – i.e. multiple unary or
virtual databases. Fortunately, the execution of these entities on different channels do
not interfere with each other: a writer on one unary channel will not cause race
conditions on another channel. Similarly, iterator in a `cesiumDB` is simply a stream
wrapper that forwards the commands it receives to its `unaryDB` internal iterators.

Out of the four entities, the only entity that requires particular attention at the
cesium level is the deleter, as we must not delete index `unaryDBs` that contain time
stamp data for other data channels. To this end, we first check under an absolute
controller region that the time range is neither controlled by a writer (in which case
we would report that there is data in that time range) nor containing any time stamp
data. If there are no data, then we call the `unaryDB`'s `delete` method to begin
deleting data. Note that there is a race condition here: right after our `HasDataFor`'s
gate is released, there could be a writer that writes some data and closes its gate
right before `delete`'s gate is acquired, causing us to delete index data for other
channels. However, we do not anticipate seeing this case often, as very rarely would one
delete an index channel but keep some data channels indexed by that index channel.

### 3.5 Conclusion

Here is a list of race conditions revealed from this RFC:

- `domainDB`:
  - `file_controller`: `acquireWriter`'s descriptor limit check may not extend into the
    following conditional statement to create a new writer.
  - Iterator: `Write` inserting a pointer before the position of interest; `Delete`
    deleting pointer(s) before or at the position of interest during operations that
    involve `reload()` (almost all operations).
  - Iterator: `Write` to the pointer at the position of the iterator; garbage collection
    of pointer at the position of the iterator during `newReader()` or `TimeRange()`.
  - Iterator: garbage collection during `Read()` from a created section reader.
  - Multiple garbage collection processes on the same file running concurrently.
- `unaryDB`:
  - Iterator: index's `domainDB` gets Garbage Collected during `Next` or `Prev`.
- `cesiumDB`:
  - Writing to an index channel while deleting that index channel with data depending on
    the said channel.
