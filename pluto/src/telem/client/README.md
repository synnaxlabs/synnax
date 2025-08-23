# Pluto Telemetry Client and Cache

This directory contains the source code for the client and caching mechanisms used for
various pluto components (line plots, valves, values, etc.) to fetch data from a Synnax
cluster. The design of this client is relatively complex, and this README is designed to
help future contributors understand it's implementation.

## Interface

The telemetry client exposes the `Client` interface. This and its various arguments and
return types are the only public facing part of the client. This is interface is a
composition of the following interfaces:

1. `ChannelClient`: Used for resolving information about a channel from its key.
2. `ReadClient`: Used for reading data from one or more channels across a specified time
   range.
3. `StreamClient`: Used for streaming live data from one or more channels.

Notice that the `Client` interface does not expose any methods for writing data to the
Synnax cluster. The client is only focused on serving and caching reads. Implentations
for writing data are defined through the `controller` module

## Internal Components

The client has five main components:

1. `Core`: The central implementation of the `Client` interfaces, which composes the
   other components in the module.
2. `Proxy`: A client implementation that wraps another client (right now always the
   `Core`) and allows for the client to be swapped out at runtime.
3. `Cache`: Held in the `cache` submodule, this is an in-memory cache that manages the
   caching of both historical and real-time data from the Synnax cluster.
4. `Reader`: The `Reader` is responsible for reading historical data from the Synnax
   cluster, and populating the cache with the data.
5. `Streamer`: The `Streamer` is responsible for streaming real-time data from the
   Synnax cluster, and populating the cache with the data.

## Caching

## Batching

## Concurrency

Streamer mutex that guards editing the listeners map
