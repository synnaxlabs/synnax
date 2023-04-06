# Context Usage in Synnax

## Prerequisites

If you're unfamiliar with what contexts are, or how they're used in Go, check out this
excellent [article](https://www.digitalocean.com/community/tutorials/how-to-use-contexts-in-go)
from Digital Ocean.

This guide is inspired by the [CockroachDB context technical notes](https://github.com/cockroachdb/cockroach/blob/ff2c4a3454a63bd74fe9a2feed53a7bdcfaf5191/docs/tech-notes/contexts.md)
originally authored by Andrei Matei.

## Summary

Contexts (`context.Context`) are used throughout the Synnax codebase, and play a critical
role in propagating cancellation signals and passing instrumentation through the call
stack. Proper context use is surrounded in both controversy and mystery. These notes aim to clarify
how we reason about contexts in Synnax, and what patterns we've intentionally chosen to
(ab)use and which we've chosen to avoid.

## Where We Use Contexts

Generally speaking, contexts are used for two purposes:

1. Cancellation signals - Freeing up resources (goroutines, network sockets, etc.) when an operation is aborted.
2. Request-scoped values - Passing request-specific information through the call stack.

At Synnax, we love the first, and strongly discourage the second. The reason is that we strongly prefer
__explicitly passing data__ through the call stack. Using `context.WithValue` embeds a dynamic, opaque value, and makes
it difficult to about how that value is transferred. This is a source of confusion for new contributors, and has the potential
to introduce subtle bugs that can't be caught by a compiler.

There are only two-cases where we use contexts for non-cancellation purposes: instrumentation and interfacing with external libraries.

## The Key Principle - Context Scope

**The usage of a context should be correlated with, and only with, a single request.**

## Struct Embedding - Why We're Ok With It

Making a case for embedding contexts in structs is a difficult one, and it stems from our preference for explicit data passing, and, specifically
when we were in the process of adding instrumentation to the key-value store interface. For the key-value read pipeline, we have the following
interface:

```
type Reader interface {
    Get(ctx context.Context, key []byte) ([]byte, error)
    NewIterator(ctx context.Context, opts IteratorOptions) Iterator
}

type Iterator interface {
    SeekGE(ctx context.Context, key []byte) bool
    SeekLT(ctx context.Context, key []byte) bool
    SeekPrefixGE(ctx context.Context, prefix []byte) bool
    SeekPrefixLT(ctx context.Context, prefix []byte) bool
    Next(ctx context.Context) bool
    Key(ctx context.Context) []byte
    Value(ctx context.Context) []byte
    Close(ctx context.Context)
}
```

Notice how we require a context both when instantiating the iterator, and when calling
any of its methods. This introduces ambiguity into how the iterator
uses the context. If we pass a different context to `NewIterator` than we do to `SeekGE`
and cancel the former, what happens when we try to execute the operation
on the latter? Did `NewIterator` start some goroutines or open a socket that was shutdown?.
This ambiguity is a continual source of confusion in many areas of the codebase.

While the above interface satisfies go best practices, it doesn't make sense. The reality is
that an iterator is almost always tied to a single operation, meaning
that the context passed to `NewIterator` is/should be the same context passed to all of its
methods. We can change our interface to reflect this:

```
type Readable interface {
    NewReader(ctx context.Context) Reader
}

type Reader interface {
    Get(key []byte) ([]byte, error)
    NewIterator(opts IteratorOptions) Iterator
}

type Iterator interface {
    SeekGE(key []byte) bool
    SeekLT(key []byte) bool
    SeekPrefixGE(prefix []byte) bool
    SeekPrefixLT(prefix []byte) bool
    Next() bool
    Key() []byte
    Value() []byte
    Close()
}
```

We add an additional `Readable` interface that constrains the usage of the `Reader`
interface to a single context. Not only is it clear that the context passed to `NewReader`
is the ONLY context used for all read operations, but it also makes it clear that the
`Reader` interface should **only** be used underneath a single context.

This pattern is even more expressive when we consider the `Writer` interface:

```
type Writable interface {
    NewWriter(ctx context.Context) Writer
}

type Writer interface {
    Set(key, value []byte) error
    Delete(key []byte) error
    Commit() error
    Close() error
}
```

Every write represents an atomic transaction on the underlying key-value store. By passing
the context to `NewWriter`, we're effectively enforcing that the transaction exist only
within the scope of the request.

In short, we think **it's ok to embed contexts in structs (or interfaces) when the lifetime of the lifetime of the struct is tightly coupled to the lifetime of the context.**

## Instrumentation

Virtually every area of the code base requires some degree of instrumentation, including logging, tracing, and metrics.
As covered in the [alamos RFC](/docs/tech/rfc/0011-230401-alamos-instrumentation.md), passing this instrumentation
explicitly through the call stack is a non-starter. The only sustainable alternative was to use `context.WithValue`.

The ubiquitous usage of instrumentation is the only reason doing this is justified.

## Interfacing with External Libraries

We use libraries (like GRPC) that use contexts for various reasons. In these cases, we have no choice but to follow their lead.
In these cases, we do everything we do everything we can to reduce the so-called "footprint" of the context's implicit usage.

# Contexts in Structs


