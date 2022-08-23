# 7 - Data Types - Adding Meaning to the Bytes

**Feature Name** - Data Types \
**Status** - Proposed \
**Start Data** - 22-08-23

# 1 - Summary

In this RFC I propose a design for managing data types across the Delta cluster.

# 2 - Motivation

We currently allow callers to write arbitrary byte slices to disk using only a
density (i.e. the number of bytes per sample). This provides flexibility for
users to define their own data types, and gives just enough context to perform
range lookups efficiently, but a density doesn't give any information on what the
bytes in a sample represent. An 8 byte sample may be a float64, uint64, timestamp,
or something else entirely. This poses a challenge when writing client libraries
that aim to provide cleanly formatted data to users. To deliver a numpy array to
a caller, we need more information on the data type.

# 3 - Design

There are four aspects we need to consider when implementing data types:

1. Flexibility - 
   * How easily can we add new data types? 
   * Can users define their own data types?
2. Information Sufficiency -  
   * How much information do we need to provide clients, agents, and internal 
   services with so that they can encode, decode, and operate on data? 
   * How often do we need to send information on a data type? 
   * Should we store the data type with each segment? With each channel? 
   * How do we propagate this information across the cluster? 
3. Validation -
   * How do we ensure that data written is valid? 
   * Where do we validate the data? Client side? Server side? If server side, 
   where on the server?
4. Complexity - 
   * How much complexity should we sacrifice in service of enabling points 1-3?

This design does not attempt to answer these questions in one fell swoop. Achieving
a sustainable type system will require continuous iteration over extended periods of time.
Instead, I focus on defining a starting point that places as few restrictions on
extension as possible.

## Potential Solutions

### 




```go
package telem

type DataType struct {
    Key string
    Density Density
}
```



