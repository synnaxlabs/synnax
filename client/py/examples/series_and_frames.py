#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np

import synnax as sy

# Construct a series from a list of numbers. In this case, the series will
# automatically be of type int64.
series = sy.Series([1, 2, 3, 4, 5])

# Construct a series from a list of numbers, but this time we specify the type
# explicitly.
series = sy.Series([1, 2, 3, 4, 5], data_type=sy.DataType.FLOAT32)

# Construct a series from a list of strings. In this case, the series will
# automatically be of type string.
series = sy.Series(["apple", "banana", "cherry"])

# Construct a series from a numpy array. This is the most efficient way to construct a
# series from a large amount of data.
series = sy.Series(np.array([1, 2, 3, 4, 5], dtype=np.float32))

# Construct a series from a list of JSON objects. This is useful when you have a series
# that has been serialized to JSON.
series = sy.Series([{"red": "cherry"}, {"yellow": "banana"}, {"orange": "orange"}])

series = sy.Series([1, 2, 3, 4, 5])

print(series[0])  # 1
print(series[-1])  # 5

series = sy.Series([1, 2, 3, 4, 5])
# Access the underlying numpy array
arr = np.array(series)
print(arr)  # [1 2 3 4 5]

series = sy.Series([1, 2, 3, 4, 5])
py_list = list(series)
print(py_list)  # [1, 2, 3, 4, 5]

start = sy.TimeStamp.now()

tr = sy.TimeRange(start, start + 5 * sy.TimeSpan.SECOND)

series = sy.Series(
    data=[1, 2, 3, 4, 5],
    data_type=sy.DataType.FLOAT64,
    time_range=tr,
)

series = sy.Series([1, 2, 3, 4, 5])
print(len(series))  # 5

string_series = sy.Series(["apple", "banana", "cherry"])
print(len(string_series))  # 3

series = sy.Series([1, 2, 3, 4, 5])
print(series.data_type)  # int64
print(series.data_type == sy.DataType.STRING)  # False

series = sy.Series([1, 2, 3, 4, 5])
print(max(series))  # 5
print(min(series))  # 1

# Construct a frame for the given channel names.
frame = sy.Frame(
    {
        "channel1": sy.Series([1, 2, 3, 4, 5]),
        "channel2": sy.Series([5, 4, 3, 2, 1]),
        "channel3": sy.Series([1, 1, 1, 1, 1]),
    }
)

# Construct a frame for the given channel keys
frame = sy.Frame(
    {
        1: sy.Series([1, 2, 3, 4, 5]),
        2: sy.Series([5, 4, 3, 2, 1]),
        # Notice that series do not need to be the same length.
        3: sy.Series([1, 1, 1]),
    }
)

# Or construct a frame with multiple series for a single channel
frame = sy.Frame(
    {
        "channel1": [
            sy.Series([1, 2, 3, 4, 5]),
            sy.Series([5, 4, 3, 2, 1]),
            sy.Series([1, 1, 1, 1, 1]),
        ],
        "channel2": [sy.Series([1, 2, 3, 4, 5])],
    }
)

frame = sy.Frame(
    {
        "channel1": [sy.Series([1, 2]), sy.Series([3, 4, 5])],
        "channel2": sy.Series([5, 4, 3, 2, 1]),
        "channel3": sy.Series([1, 1, 1, 1, 1]),
    }
)

multi_series = frame["channel1"]
# Access a value
print(multi_series[0])  # 1

# Access a value from a specific series
print(multi_series.series[0][0])  # 1

# Construct a Python list from the MultiSeries
py_list = list(multi_series)
print(py_list)  # [1, 2, 3, 4, 5]

frame = sy.Frame(
    {
        "channel1": sy.Series([1, 2, 3, 4, 5]),
        "channel2": sy.Series([5, 4, 3, 2, 1]),
        "channel3": sy.Series([1, 1]),
    }
)

obj = frame[3]
print(obj)  # {'channel1': 4, 'channel2': 2, 'channel3': None}

# Convert the frame to a pandas DataFrame
df = frame.to_df()
print(df)
