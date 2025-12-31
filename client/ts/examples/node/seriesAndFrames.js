// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Series } from "@synnaxlabs/client";

// Construct a series from an array of numbers. In this case, the series will
// automatically be of type float64.
let series = new Series([1, 2, 3, 4, 5]);

// Construct a series from an array of numbers, but this time we specify the type
// explicitly.
series = new Series({ data: [1, 2, 3, 4, 5], dataType: "float32" });

// Construct a series from an array of strings. In this case, the series will
// automatically be of type string.
series = new Series(["apple", "banana", "cherry"]);

// Construct a series from a Float32Array. This is the most efficient way to construct a
// series from a large amount of data.
series = new Series(new Float32Array([1, 2, 3, 4, 5]));

// Construct a series from a JSON object. This is useful when you have a series that has
// been serialized to JSON.
series = new Series([{ red: "cherry" }, { yellow: "banana" }, { orange: "orange" }]);

series = new Series([1, "a", 3, "b", 5]);

series = new Series([1, 2, 3, 4, 5]);

console.log(series.at(0)); // 1
console.log(series.at(-1)); // 5

series = new Series([1, 2, 3, 4, 5]);
// Is it a number? Is it a string? Who knows?
let v = series.at(0);

series = new Series([1, 2, 3, 4, 5]);
let easierSeries = series.as("number");
// Now we have a guarantee that this is a series of numbers.
v = easierSeries.at(0);
console.log(v);

series = new Series({ data: [1, 2, 3, 4, 5], dataType: "int8" });
const ta = series.data;
console.log(ta); // Int8Array [ 1, 2, 3, 4, 5 ]

series = new Series([1, 2, 3, 4, 5]);
let jsArray = [...series];
console.log(jsArray); // [ 1, 2, 3, 4, 5 ]
const jsArray2 = Array.from(series);
console.log(jsArray2); // [ 1, 2, 3, 4, 5 ]

series = new Series([{ red: "cherry", yellow: "banana", orange: "orange" }]);
jsArray = [...series];
console.log(jsArray); // [ { red: 'cherry', yellow: 'banana', orange: 'orange' } ]

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/client";

const start = TimeStamp.now();

const tr = new TimeRange(start, start.add(TimeSpan.seconds(5)));

series = new Series({
  data: [1, 2, 3, 4, 5],
  dataType: "float64",
  timeRange: tr,
});

series = new Series([1, 2, 3, 4, 5]);
console.log(series.length); // 5

const stringSeries = new Series(["apple", "banana", "cherry"]);
console.log(stringSeries.length); // 3

import { DataType } from "@synnaxlabs/client";

series = new Series([1, 2, 3, 4, 5]);
console.log(series.dataType.toString()); // "float64"
console.log(series.dataType.equals(DataType.FLOAT64)); // true

series = new Series([1, 2, 3, 4, 5]);
console.log(series.max); // 5
console.log(series.min); // 1
console.log(series.bounds); // { lower: 1, upper: 5 }

import { Frame } from "@synnaxlabs/client";

// Construct a frame for the given channel names.
let frame = new Frame({
  channel1: new Series([1, 2, 3, 4, 5]),
  channel2: new Series([5, 4, 3, 2, 1]),
  channel3: new Series([1, 1, 1, 1, 1]),
});

// Construct a frame for the given channel keys
frame = new Frame({
  1: new Series([1, 2, 3, 4, 5]),
  2: new Series([5, 4, 3, 2, 1]),
  // Notice that series do not need to be the same length.
  3: new Series([1, 1, 1]),
});

// Construct a frame from a map
frame = new Frame(
  new Map([
    ["channel1", new Series([1, 2, 3, 4, 5])],
    ["channel2", new Series([5, 4, 3, 2, 1])],
    ["channel3", new Series([1, 1, 1, 1, 1])],
  ]),
);

// Or from an array of keys and series
frame = new Frame(
  ["channel1", "channel2", "channel3"],
  [
    new Series([1, 2, 3, 4, 5]),
    new Series([5, 4, 3, 2, 1]),
    new Series([1, 1, 1, 1, 1]),
  ],
);

// Or construct a frame with multiple series for a single channel
frame = new Frame({
  channel1: [
    new Series([1, 2, 3, 4, 5]),
    new Series([5, 4, 3, 2, 1]),
    new Series([1, 1, 1, 1, 1]),
  ],
  channel2: [new Series([1, 2, 3, 4, 5])],
});

frame = new Frame({
  channel1: [new Series([1, 2]), new Series([3, 4, 5])],
  channel2: new Series([5, 4, 3, 2, 1]),
  channel3: new Series([1, 1, 1, 1, 1]),
});

const multiSeries = frame.get("channel1");
// Access a value
console.log(multiSeries.at(0)); // 1

// Access a value from a specific series
console.log(multiSeries.series[0].at(0)); // 1

// Construct a Javascript array from the MultiSeries
jsArray = [...multiSeries];
console.log(jsArray); // [ 1, 2, 3, 4, 5 ]

frame = new Frame({
  channel1: new Series([1, 2, 3, 4, 5]),
  channel2: new Series([5, 4, 3, 2, 1]),
  channel3: new Series([1, 1]),
});

let obj = frame.at(3);
console.log(obj); // { channel1: 1, channel2: 5, channel3: undefined }

frame = new Frame({
  channel1: new Series([1, 2, 3, 4, 5]),
  channel2: new Series([5, 4, 3, 2, 1]),
  channel3: new Series([1, 1]),
});
try {
  obj = frame.at(3, true); // Throws an error
} catch (e) {
  console.log(e.message); // no value at index
}
