# 1 - Cesium - Channel Segment Storage Engine

**Feature Name** - Channel Segment Storage Engine <br />
**Status** - Draft <br />
**Start Date** - 2024-03-10 <br />
**Authors** Leo Liu <br />

# 0 - Summary

In this RFC I discuss the nuances of Cesium's concurrency. More specifically, how it handles race conditions.

# 1 - Vocabulary

**Thread** – An execution of program independent of others. In Go, a thread is implemented as a goroutine, which is more lightweight and compact.
**Concurrency** - Multiple threads(goroutines in the context of Go) running at the same time in parallel
**Mutex** - Mutual Exclusion, i.e. only one running at once
**Deadlock** - When two threads wait for each other to finish running or to release some Lock, therefore become stuck forever

# 2 - Motivation

Cesium-based atomic clocks are "the most accurate realization of a unit that mankind has yet achieved." By extension, Cesium, the database named after the element in question, must also be accurate. This is challenging, however, due to Cesium's concurrent nature. When multiple subprocesses(goroutines) are to apply operations on the database, it is vital to handle these potentially conflicing operations in an orderly and predictable manner. 

As Cesium looks to become production-ready, this task becomes more important as Cesium must support Read operations, Write operations, and Delete operations (through a tombstone/garbage-collection system). All of the operations must coexist without conflicting each other.

# 4- Design

The core method through which concurrency is protected is the *Mutex*.

The Mutex allows for operations of threads (Going forward, I will refer to these as goroutines in the specific context as go) to be MUTually EXclusive. In other words, if one goroutine acquires a mutx (by calling `mu.Lock()`, all other goroutines trying to acquire it must wait until it is freed, i.e. `mu.Lock()` will block until the first goroutine releases the Mutex, i.e. `mu.UnLock()`. In addition, one goroutine may acquire a Read-Lock only (`mu.RLock()`), which allows other goroutines to acquire Read-Locks, but no goroutine may acquire a full-Lock. 

Mutexes are present throughout Cesium. We will start discussing them at the most fundamental level – the domain. The domain holds a struct `index` which contains a struct struct called `mu` which contains a Mutex and a slice called `pointers`. These pointers store information on where the different domains (segments) of data for Cesium is stored (including filekey, offset, length, etc.) As you can imagine, we **really want `pointers` to be thread-safe**. It would be pretty disasterous if multiple goroutines were to modify it at the same time. Currently, as Cesium is implemented, each method that needs to read or write `index` acquires a Read or Write lock in its method body, while it is being called on the `idx` object. As we come to realize, this causes problems.

Consider the operation of deleting a timerange at a Unary level. Since a domain database is index-agnostic (i.e. it does not know if its overlaying Unary DB's timestamps are regular or set by some other Unary DB), this deletion operation must be initiated at the Unary level in order to get the correct pointers from where we need to delete. To do this, we must invoke multiple functions from the domainDB that either read from or write to the index. This is where the problem arises. Since each of these operations ensure their own thread-safety, i.e. they acquire a Lock for themself, as we previously established, we cannot acquire an overarching Lock, as each individual operation would fail to acquire a Lock (a Lock cannot be acquired while another goroutine holds the Lock – *even if that goroutine is the thread itself*). Because we cannot acquire an overarching Lock, we cannot reasonably ensure that between these individual operations, the data in `index` remains the same – it could totally be written to or deleted from!

Since we cannot ascertain ourselves the integrity of the data without an overarching Lock, nor can we acquire that overarching Lock, we must change the architecture of the index. As outlined, there are but two ways to address this: either we remove each individual operation's blocks, or we somehow make sure that no changes can be made between operations. 
