# 1 - Cesium - Channel Segment Storage Engine

**Feature Name** - Channel Segment Storage Engine <br />
**Status** - Draft <br />
**Start Date** - 2024-03-10 <br />
**Authors** Leo Liu <br />

# 0 - Summary

In this RFC I discuss the nuances of Cesium's concurrency. More specifically, how it handles race conditions across its different operations. The motivating example is a design problem faced by implementing the deletion operation – I first introduce a solution to that problem, and then expand to talk about Cesium as a whole and examine the potential race conditions.

# 1 - Vocabulary

**Thread** – An execution of program independent of others. In Go, a thread is implemented as a goroutine, which is more lightweight and compact.

**Concurrency** - Multiple threads(goroutines in the context of Go) running at the same time in parallel

**Mutex** - Mutual Exclusion, i.e. only one running at once

**Deadlock** - When two threads wait for each other to finish running or to release some Lock, therefore become stuck forever

# 2 - Motivation

Cesium-based atomic clocks are "the most accurate realization of a unit that mankind has yet achieved." By extension, Cesium, the database named after the element in question, must also be accurate. This is challenging, however, due to Cesium's concurrent nature. When multiple subprocesses(goroutines) are to apply operations on the database, it is vital to handle these potentially conflicing operations in an orderly and predictable manner.

As Cesium looks to become production-ready, this task becomes more important as Cesium must support Read operations, Write operations, and Delete operations (through a tombstone/garbage-collection system). All of the operations must coexist without conflicting each other.

The core method through which concurrency is protected is the *Mutex*.

The Mutex allows for operations of threads (Going forward, I will refer to these as goroutines in the specific context as go) to be MUTually EXclusive. In other words, if one goroutine acquires a mutex (by calling `mu.Lock()`, all other goroutines trying to acquire it must wait until it is freed, i.e. `mu.Lock()` will block until the first goroutine releases the Mutex, i.e. `mu.UnLock()`. In addition, one goroutine may acquire a Read-Lock only (`mu.RLock()`), which allows other goroutines to acquire Read-Locks, but no goroutine may acquire a full-Lock.

Mutexes are present throughout Cesium. We will start discussing them at the most fundamental level – the domain. The domain holds a struct `index` which contains a struct called `mu` consisting of a Mutex and a slice called `pointers`. These pointers store information on where the different domains (segments) of data for Cesium is stored (including filekey, offset, length, etc.) As one might imagine, we **really want `pointers` to be thread-safe**. It would be pretty disasterous if multiple goroutines were to modify it at the same time. Currently, as Cesium is implemented, each method that needs to read or write `index` acquires a Read or Write lock in its method body, while it is being called on the `idx` object. As we come to realize, this causes problems.

Consider the operation of deleting a timerange at the Unary level. Since a domain database is index-agnostic (i.e. it does not know if its overlaying Unary DB's timestamps are regular or set by some other Unary DB), this deletion operation must be initiated at the Unary level in order to get the correct pointers from where we need to delete. To do this, we must invoke multiple functions from the domainDB that either read from or write to the index. This is where the problem arises. Since each of these operations ensure their own thread-safety, i.e. they acquire a Lock for themself, as we previously established, we cannot acquire an overarching Lock, as each individual operation would fail to acquire a Lock (a Lock cannot be acquired while another goroutine holds the Lock – *even if that goroutine is the thread itself!*). Because we cannot acquire an overarching Lock, we cannot reasonably ensure that between these individual operations, the data in `index` remains the same – it could totally be written to or deleted from!

Since we cannot ascertain ourselves the integrity of the data without an overarching Lock, nor can we acquire that overarching Lock, we must change the architecture of the index.

# 4- Design

## 4.0 Index Changes

To address this, we will make the `index` methods that may require an `RLock` take in `withLock` as a parameter – this way, callers may choose to acquire a lock for their current thread, or not if they already have one, such as the example mentioned in section 2. This design was chosen over removing acquiring locks in `index` methods as a whole since it maintains the autonomy of these methods to acquire locks while offloading the responsibility of callers to acquire locks, thus improving thread-safety.

We will also implement an additional field in the `domain/iterator` interface called `locked`. For a duration, an iterator could be locked, i.e. all of its operations on indices will no longer acquire their own locks. The lock acquired by this iterator is automatically relinquished either on `iterator.Unlock()` or on `iterator.Close()`. This way, we can maintain the lock for the whole duration of the delete operation without running into deadlocks.

## 4.1 High level Analysis of Race Conditions in cesium

There are but three entities that are at the bottom of all Cesium operations: `Iterator`, `Writer`, and `Writer` used for deletion (I will call this a `Deleter` from now on).

Therefore, there will be potential race conditions whenever we have those operations working in tandem: when an `Iterator` is reading while a `Writer` is writing, when a `Deleter` is deleting a domain read by an `Iterator` – also, the same operation by two entities will also cause chaos – like in a case where a `Deleter` is trying to delete something currently written to by `Writer`.

There are many different mutexes in Cesium:

- cesium/db.mu: this mutex protects the map that stores individual `unary` and `virtual` db's in Cesium.
- domain/idx.mu: this mutex protects the slices of pointers and tombstones in each domainDB that specifies what chunks of data  are stored where on the disk.
- domain/file_controller/writers.mu: this mutex is part of the file controller, and protects the map of open IO writer instances to files.
- domain/file_controller/readers.mu: this mutex is part of the file controller, and protects the map of open IO reader instances to files.
- controller/controller.mu: this mutex is part of the controller (different from file controller!), and protects the slice in controller that stores the regions opened on the controllers.
- controller/region.mu: this mutex is also part of the controller, and protects the set that stores the open gates on each region.

These mutexes are designed to provide protection for the various concurrent data in synnax.

## 4.2 Race Conditions at the Domain level

We will start by discussing race conditions at the lowest level of the database architecture – domain. The `domain/db` interacts directly with the file system and is not indexed by anything. Its job description is very simple – it tracks where data are and are not through `pointers` and `tombstones`, and reads and writes data there through reader and writer entities managed by the `domain/file_controller`.

### 4.2.1 Write at the Domain level

The `Write` function in `domain` writes a `[]byte` data to the domain. This is done through the following steps:

1. A new `domain/writer` is created
   1. A TrackedWriteCloser is acquired from the `domain/file_controller`
      1. To do so, we `RLock` the `domain/file_controller_writers` (the map of file names to open writers) and iterate through it to see if they are 1) under capacity 2) not currently being written to. If we can acquire a such reader, then we release the lock and return the writer. Note that the flag indicating whether a writer is currently writing is an `atomic.Bool`, therefore we can ensure its thread safety so no two writers will ever write to the same file at the same time.

      2. Otherwise, meaning we cannot acquire any of the currently open writers, we will create a new writer if we are not yet at the descriptor limit. Note that the `RLock` is released prior to this step, meaning between the time we create the new writer, more writers could be added to the map – this is fine, since we would not want those write requests to interfere with the one we are currently processing.

      3. If we cannot create a new writer since we are at `DescriptorLimit`, then we try to garbage collect the writers there are available and wait until a writer gets released – either through garbage collection or through one of the writers closing, and we recurse the function again to acquire a writer. Note that apart from going through the map, there is no lock on the `writers` map. This is fine since the state of the file controller that are needed between operations (i.e. in creating a new writer or garbage collect) – like the acquired status of controllers or the counter – are atomic.
   2. A call to `db.idx.overlap` is made to check whether the current writer's domains overlaps with an existing domain (pointer) in the database. This call is called with `withLock=true`, meaning that while checking for overlaps, no other pointers may write to `pointers`. If there is an overlap an error is returned. Note that this is prone to race conditions since even though there might be no overlaps by the time this overlap check is made, other writers could write to `pointers` after the check is complete, resulting in the two writers writing to the same domain. This is tested via a test in domain/race_test.go.
   3.
