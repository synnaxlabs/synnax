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

// Construct a series from a Float32Array. This is the most efficient way to
// construct a series from a large amount of data.
series = new Series(new Float32Array([1, 2, 3, 4, 5]));

// Construct a series from a JSON object. This is useful when you have a series
// that has been serialized to JSON.
series = new Series([{ red: "cherry" }, { yellow: "banana" }, {orange: "orange" }]);

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

import { TimeRange, TimeStamp, TimeSpan } from "@synnaxlabs/client";

const start = TimeStamp.now();

const tr = new TimeRange(start, start.add(TimeSpan.seconds(5)));

series = new Series({
  data: [1, 2, 3, 4, 5],
  dataType: "float64",
  timeRange: new TimeRange({ start, end: start.add(TimeSpan.seconds(6)) })
});