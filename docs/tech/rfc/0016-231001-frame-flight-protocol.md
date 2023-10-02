# 16 - Frame Flight Protocol

**Feature Name**: Frame Flight Protocol <br />
**Start Date**: 2023-10-01 <br />
**Authors**: Wynn Kaza <br />
**Status**: Started <br />
**Deadline**: First: 10/20/2023 : Minimum Example in C++,Typescript

# 0 - Summary

A key aspect of data transfers is reducing the amount of data transferred over a network,
while maintaining accuracy. The current system sends [Frames](./0010-230104-frame-spec.md)
over the system, which can force unnecessary metadata to be sent over the network. The
proposed improvement changed is to take advantage of known traits of the frame to reduce
the amount of data sent over the network. The final result with be a byte array that can be
unpacked at the receiver, and thus recreate the frame.

# 1 - Vocabulary

**Frame** - Architecture for wrapping primitive arrays with identifying
meta-data into aligned arrays that streamline the telemetry transfer process.
**Telemetry** - Data samples received from sensors and sent to actuators; typically
stored on Synnax server. More details available [here](../../../pluto).
**Series** - A strongly typed collection of telemetry samples over a time range. The
fundamental unit of data transfer in Synnax server.

# 2 - Motivation

A key feature of Synnax is its ability to send real-time data to be analyzed by the system.
The current system utilized Synnax Frames, which are organized as JSON, and thus add a heavy
workload to the network. Furthermore, JSON is encoded in utf-8, thus having a higher overhead
cost per datapoint than other encoding schemes. As both the transmitter and receiving side of the network should have
knowledge of the system design before transmission, the goal is to create a library capable of
reducing the JSON data to a byte array, and thus remove much of the overhead cost of sending
JSON data.

This system will be implemented in four different languages (C++, Go, Python, TypeScript) to
meet the various demands of Synnax.

# 5 - Detailed Design

## 5.0 - Dynamic Analysis of Algorithm

Understanding key similarities within the Frame is essential to discover places to reduce
the overhead of JSON. Within the implementation, the idea is to find information that could be
sent in a header, or in a meta-data only packet at initialization. Some important things to test for
are
- All data arrays are the same length
- Whether partial keys or all keys are being sent
- Whether the time range is strongly/weakly aligned
Based on flags which check for each of these, we can reduce the amount of data sent at any period in time.
For example, if you know that all the data arrays will have the same length, we can send the length at the start
of the byte array, and apply it to all the series. Similarly, for partial keys and time range, if we know that
all the series share a similarity, we can send this data at the beginning of the packet, instead of having to include
it as part of every series sent.
```json
keys: ["one", "two", "three", "four"]
series: {
{
    "DataType": int8,
    "Data": [1, 2, 3, 4, 5],
    "TimeRange": [5, 10]
},
{
    "DataType": int16,
    "Data": [20, 15, 12, 19, 32],
    "TimeRange": [5, 10]
},
{
    "DataType": int8,
    "Data": [1, 2, 3, 4, 5, 6],
    "TimeRange": [5, 10]
},
{
    "DataType": int16,
    "Data": [20, 15, 12, 19, 32],
    "TimeRange": [5, 10]
},
}
```


### 5.0.0 - Data Layout

Some important factors to consider are that **keys** are represented as a **uint32**. The **Time Range** is represented with
a start and end time, both of which are **uint64**. Finally, the **data array size** will currently be bound by a **uint32** but this
may be changed in further iterations if discovered to be unnecessary and costly.

The first byte of every frame (represented as a byte array) will contain the flags for the array. Currently, the three flags are
listed above: Equal Data Size, Strongly Aligned Timestamp, All Channels Requested
Therefore, the first byte will look like the following
```python
[0, 0, 0, 0, 0, equalDataSize, stronglyAlignedTimestamp, allChannels]
```

If the **Equal Data Size Flag** is set to true, the next 4 bytes will include the size representing the size of all Series data arrays. </br>
If the **Strongly Aligned Timestamp Flag** is set to true, the following 16 bytes will include information about the timestamp for all Series arrays,
with the startTime going first, and the endTime going second </br>
If the **allChannels** is set to true, nothing will be different </br>

All conditions above are read from left to right, for example, if both the **Equal Data Size Flag** and **Strongly Aligned Timestamp Flag**
are set, then the 4 bytes following the first byte would include the **Equal Data Size**, then the following 16 bytes would include timestamp
information.

For the rest of the byte array, the following rules apply </br>
- Iterate through series sequential (0, 1, 2, ... n)
- If **Equal Data Size Flag** is not set, then the first four bytes should include the size of the data array
- If **All Channels Flag** is not set, then the next four bytes include the uint32 key for the designated series
- Then, all values within the data array should be sent
- If **Strongly Aligned Timestamp Flag** is not set, then the next 16 bytes should include timestamp information

### 5.0.1 - Decoding

TODO!!!
