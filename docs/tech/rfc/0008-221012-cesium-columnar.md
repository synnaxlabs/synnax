# 8 - Columnar Storage for Irregular Data

- **Feature Name** - Cesium - Columnar Storage for Irregular Data
- **Status** - Complete
- **Start Date** - 2022-10-12
- **Authors** - Emiliano Bonilla

# 0 - Summary

In this RFC I propose a redesign of Cesium's architecture in order to support irregular
time series data. This is a major internal redesign, but it still aims to maintain a
similar external API that should result in minimal changes to the user experience.

# 1 - Vocabulary

# 2 - Motivation

Cesium works quite well for irregular time-series data, but as we start to roll out
Synnax to more users, we're starting to see more and more requests for handling
irregularly sampled data. In many cases, a lock of support for irregular data is a deal
breaker for users.

# 3 - Design

This design extends and modifies the existing Cesium architecture presented in this
[rfc](https://github.com/synnaxlabs/delta/blob/main/docs/rfc/0001-220517-cesium-segment-storage.md).
The new architecture (V0.2) introduces the concept of 'indexes' which provide alignment
between timestamp values and data stored in a segment.

## 3.0 - Indexes

Indexes define alignments between timestamps and stored segments. There are three types
of indexes: the root index, channel based indexes, and rate based indexes. When
explaining the indexes, it's useful to think about Cesium's data model as a very large
excel spreadsheet.

### 3.0.0 - Root Index

The root index can be though of as the row # in the spreadsheet. It's a channel that
represents the `array index` of a particular value. It's useful to think of it as a
column of ordered nanosecond timestamps. The root index is used as an intermediary
translator between other indexes (i.e. channel or rate based indexes) and the actual
data stored in the segments. When a segment is written to disk, it contains an
`Alignment` field that defines the position of the first value in the segment relative
to the root index.

### 3.0.1 - Fixed Rate Index

Fixed rate indexes are what was used in the original Cesium design. They assume that
data is sampled at a fixed rate. To translate between a timestamp and a position on the
root index, the following formula is used:

```
root_index_position = time_stamp / period_of_data_rate
```

To make this more concrete, let's say we have a channel that is sampled at 10Hz, and
we're writing ten values to disk starting at timestamp 10 seconds. The data would
resemble the following:

| Root Index | Time Stamp | Value |
| ---------- | ---------- | ----- |
| 100        | 10s        | 0     |
| 101        | 10.1s      | 1     |
| 102        | 10.2s      | 2     |
| 103        | 10.3s      | 3     |
| 104        | 10.4s      | 4     |
| 105        | 10.5s      | 5     |
| 106        | 10.6s      | 6     |
| 107        | 10.7s      | 7     |
| 108        | 10.8s      | 8     |
| 109        | 10.9s      | 9     |

If we were to write another five sample segment starting at timestamp 11.5s, the data
would be appended as follows:

| Root Index | Time Stamp | Value |
| ---------- | ---------- | ----- |
| 100        | 10s        | 0     |
| 101        | 10.1s      | 1     |
| 102        | 10.2s      | 2     |
| 103        | 10.3s      | 3     |
| 104        | 10.4s      | 4     |
| 105        | 10.5s      | 5     |
| 106        | 10.6s      | 6     |
| 107        | 10.7s      | 7     |
| 108        | 10.8s      | 8     |
| 109        | 10.9s      | 9     |
| 115        | 11.5s      | 10    |
| 116        | 11.6s      | 11    |
| 117        | 11.7s      | 12    |
| 118        | 11.8s      | 13    |
| 119        | 11.9s      | 14    |

### 3.0.2 - Channel Based Index

Channel based indexes use a channel to align timestamps with the root index. This
channel is called the `index channel`. The index channel contains ordered timestamps. To
translate between a timestamp and a position on the root index, a lookup is performed on
the index channel.

Indexed channels are useful for irregularly sampled data. A user can create an index
channel, write timestamp values to it, and then write to the data channel.

As an example, let's say we have a timestamp and temperature sensor channel that starts
at timestamp 10s:

| Root Index  | Time Stamp | Temperature |
| ----------- | ---------- | ----------- |
| 10000000000 | 10s        | 0           |
| 10000000001 | 13s        | 1           |
| 10000000002 | 14s        | 2           |
| 10000000003 | 16.5s      | 3           |
| 10000000004 | 17s        | 4           |
| 10000000005 | 19s        | 5           |
| 10000000006 | 22s        | 6           |
| 10000000007 | 24s        | 7           |

It might seem strange that the root index is a nanosecond timestamp, but this is
necessary to support the maximum possible resolution of the index channel. If the index
channel were to sample once every nanosecond, the root index would have a direct
correlation to the index channel.

:::caution When looking at the three tables above, it's important to notice that values
that share the same root index position do not necessarily share the same timestamp. :::

## 3.1 - Channel Types

With the introduction of indexes, channels are now split into three separate types:

### 3.1.0 - Index Channels

Index channels store ordered int64 timestamp values. They are used to align timestamps
with the root index. Extensive validation is performed on index channels, which means
write performance is the slowest of the three channel types. Index channels must be
written to before any data channels that use the index.

### 3.1.1 - Indexed Data Channels

Indexed data channels are data channels that rely on an index channel to align
timestamps with the root index. Validation is performed on indexed data channels to
ensure that timestamps are properly defined for all values, which means write
performance is slower that rate based data channels. Indexed data channels must be
written to after the index channel that they use.

### 3.1.2 - Rate Based Data Channels

Rate based data channels are data channels that rely on a fixed rate to align timestamps
with the root index. Because timestamps can be resolved based on a fixed data rate,
validation is not performed on rate based data channels, which means write performance
is the fastest of the three channel types. Read performance is also the fastest of the
three channel types, because no lookup is required to resolve timestamps.

## 3.2 - Implementing Channel Based Indexes

## 3.3 - Segment Validation

### 3.3.0 - Index Segment Validation

1. Data must be ordered int64 values.
2. The `Start` timestamp of a segment must be equal to the first value in its data.
3. Cannot overlap with other segments.

### 3.3.1 - Indexed Segment Validation

1. The index must be defined and contiguous for all values in the segment.
2. The `Start` timestamp of the segment must exactly align with a timestamp in the
   index.
3. Cannot overlap with other segments.

### 3.3.2 - Rate Segment Validation

1. Cannot overlap with other segments.
