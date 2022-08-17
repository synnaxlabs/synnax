<br />
<p align="center">
    <a href="https://aryaanalytics.com/">
        <img src="docs/media/icon-full-title-white.png" width="70%"/>
    </a>
</p>

# Cesium

Cesium extends CockroachDB's [Pebble](https://github.com/cockroachdb/pebble) to provide fast storage for time-series
data.

## Use Case

Cesium is tailored towards a specific class of time-series data:

1. Regular - samples are taken at specific, known intervals. Cesium will not work well with data that arrives at
   unpredictable
   intervals.
2. High Speed - cesium is happiest at sample rates between 10 Hz and 1 MHz. Although it can work with data at any rate,
   it will be far slower than other storage engines for low sample rates.

## Concepts 

The following is an overview of Cesium concepts from an interface perspective. A detailed design 
[RFC](https://github.com/arya-analytics/delta/blob/main/docs/rfc/220517-cesium-segment-storage.md) is available for those interested in the implementation.

### Channels

A channel (`cesium.Channel`) is a time-ordered collection of samples. It's best to approach them as a device (whether
physical or virtual)
that emits values with the following properties:

1. Time-ordered
2. Time-unique - no two values are emitted at the same time.
3. Constant size - all emitted values have the same amount of bytes.
4. Regular - all values are emitted at a constant interval.

These values are known as samples. Samples can be measurements from a sensor, events from a stream, metrics from a
database, or images from a camera (to name a few).

### Segments

A segment (`cesium.Segment`) is a contiguous run of samples (between 1 B and 2.5 MB). Cesium stores all values in a
segment sequentially on disk, so it's naturally best to write large segments over small segments.

This obviously has implications in terms of durability, as smaller segments will prevent data loss in the event of a
failure. It's up to you to weigh the performance vs. durability risk.

### Data Types

Cesium samples can be of any data type that can be serialized to a byte array. Cesium has a few built-in data types:

```
cesium.Float64
cesium.Float32
cesium.Int64
cesium.Int32
cesium.Int16
cesium.Int8
cesium.Uint64
cesium.Uint32
cesium.Uint16
cesium.Uint8
```

### Custom Data Types

Defining a custom data type is as simple as defining a constant of type `cesium.DataType` with its size in bytes:

```go
package main

import (
	"github.com/arya-analytics/cesium"
)

// TenByTenImage is a custom data type where each sample is 10 * 10 * 3 bytes in size.
const TenByTenImage cesium.DataType = 10 * 10 * 3
```

## Production Readiness

Cesium is in Alpha state, and is not ready for production use.

## Installation

```bash
go get github.com/arya-analytics/cesium
```

## Getting Started

### Writing Samples

The following example is a simple, synchronous example of writing samples to a channel.

```go
package main

import (
	"context"
	"github.com/arya-analytics/cesium"
	"log"
)

func main() {
	ctx := context.Background()

	// Open a DB whose files live in the "testdata" directory.
	db, err := cesium.Open("testdata")
	if err != nil {
		log.Fatal(err)
	}

	const (
		dataType = cesium.Float64
		dataRate = 5 * cesium.Hz
	)

	// Create a new channel whose samples are float64 values recorded at 5 Hz.
	ch, err := db.NewCreateChannel().WithType(dataType).WithRate(dataRate).Exec(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Create a new Segment to write. If you don't know what segments are, 
	// check out the Segment documentation.
	segments := []cesium.Segment{
		{
			ChannelKey: ch.Key,
			Start:      cesium.Now(),
			Data:       cesium.MarshalFloat64([]float64{1.0, 2.0, 3.0}),
		},
	}

	// Open the query. DB.Sync is a helper that turns a typically async write 
	// into an acknowledged, synchronous write.
	if err := db.Sync(ctx, db.NewCreate().WhereChannels(ch.Key), &segments); err != nil {
		log.Fatal(err)
	}
}
```

