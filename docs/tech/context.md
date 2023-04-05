# Context Usage in Synnax

# Prerequisites

If you're unfamiliar with what contexts are, or how they're used in Go, check out this
excellent [article](https://www.digitalocean.com/community/tutorials/how-to-use-contexts-in-go)
from Digital Ocean.

This guide is inspired by the [CockroachDB context technical notes](https://github.com/cockroachdb/cockroach/blob/ff2c4a3454a63bd74fe9a2feed53a7bdcfaf5191/docs/tech-notes/contexts.md)
originally authored by Andrei Matei.

# Summary

Contexts (`context.Context`) are used throughout the Synnax codebase, and play a critical
role in propagating cancellation signals and passing instrumentation through the call
stack.

Proper context use is surrounded in both controversy and mystery. These notes aim to clarify
how we reason about contexts in Synnax, and what patterns we've intentionally chosen to
(ab)use and which we've chosen to avoid.

# The Two Reasons to Use Contexts

Generally speaking, contexts are used for two purposes:

1. Cancellation signals - Freeing up resources (goroutines, network sockets, etc.) when an operation is aborted.
2. Request-scoped values - Passing request-specific information through the call stack.

At Synnax, we love the first usage, and strongly discourage the second. Our reasoning is that we strongly prefer
__explicitly passing data__ through the call stack. Using `context.WithValue` embeds a dynamic, opaque value, and makes
it difficult to reason about what information is being passed around. This is a source of confusion for new contributors,
and has the potential to introduce subtle bugs that can't be caught by a compiler.

There are two-cases where we use contexts for anything other than cancellation signals.

## Instrumentation

Virtually every area of the code base requires some degree of instrumentation, including logging, tracing, and metrics.
As covered in the [alamos RFC](/docs/tech/rfc/0011-230401-alamos-instrumentation.md), passing this instrumentation
explicitly through the call stack is a non-starter. The only sustainable alternative was to use `context.WithValue`.

The ubiquitous usage of instrumentation is the only reason doing this is justified.

## Interfacing with External Libraries

We use libraries (like GRPC) that use contexts for various reasons. In these cases, we have no choice but to follow their lead.
In these cases, we do everything we do everything we can to reduce the so-called "footprint" of the context's implicit usage.

# The Key Principle - Context Scope

The usage of a context should be correlated with, and only with, a single operation.

# Contexts in Structs


