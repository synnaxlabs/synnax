package cesium

import "github.com/arya-analytics/x/alamos"

const (
	// createMetricsKey is the key used to store create metrics in cesium's
	// alamos.Experiment.
	createMetricsKey = "create"
	// retrieveMetricsKey is the key used to store retrieve metrics in cesium's
	// alamos.Experiment.
	retrieveMetricsKey = "retrieve"
)

// createMetrics is a collection of metrics tracking the performance and health of
// cesium's Create related operations.
type createMetrics struct {
	// dataWrite tracks the duration it takes to flush segment data to disk.
	dataWrite alamos.Duration
	// headerFlush tracks the duration it takes to flush segment kv data.
	headerFlush alamos.Duration
	// totalFlush tracks the duration it takes all the operations to disk.
	// (dataWrite,headerFlush,seeks, etc.)
	totalFlush alamos.Duration
	// lockAcquire tracks the duration it takes to acquire the lock on the channels
	// that are being written to.
	lockAcquire alamos.Duration
	// lockRelease tracks the duration it takes to release the lock on the channels
	// that are being written to.
	lockRelease alamos.Duration
	// segSize tracks the Size of each segment created.
	segSize alamos.Metric[int]
	// request tracks the total duration that the Create query is open i.e. from
	// calling Create.Stream(ctx) to the close(res) call.
	request alamos.Duration
}

type retrieveMetrics struct {
	// kvRetrieve is the time spent retrieving segment metadata from key-value storage.
	kvRetrieve alamos.Duration
	// dataRead is the duration spent reading segment data from disk.
	dataRead alamos.Duration
	// segSize tracks the Size of each segment retrieved.
	segSize alamos.Metric[int]
	// segCount tracks the number of segments retrieved.
	segCount alamos.Metric[int]
	// request tracks the total duration that the Retrieve query is open i.e. from calling Retrieve.Stream(ctx) to
	// an internal close(res) call that represents the query is complete.
	request alamos.Duration
}

func newCreateMetrics(exp alamos.Experiment) createMetrics {
	sub := alamos.Sub(exp, createMetricsKey)
	return createMetrics{
		segSize: alamos.NewGauge[int](
			sub,
			alamos.Debug,
			"segmentSize",
		),
		lockAcquire: alamos.NewGaugeDuration(
			sub,
			alamos.Debug,
			"lockAcquireDuration",
		),
		lockRelease: alamos.NewGaugeDuration(
			sub,
			alamos.Debug,
			"lockReleaseDuration",
		),
		dataWrite: alamos.NewGaugeDuration(
			sub,
			alamos.Debug,
			"dataWriteDuration",
		),
		headerFlush: alamos.NewGaugeDuration(
			sub,
			alamos.Debug,
			"headerFlushDuration",
		),
		totalFlush: alamos.NewGaugeDuration(
			sub,
			alamos.Debug,
			"totalFlushDuration",
		),
		request: alamos.NewGaugeDuration(
			sub,
			alamos.Debug,
			"requestDuration",
		),
	}
}

func newRetrieveMetrics(exp alamos.Experiment) retrieveMetrics {
	sub := alamos.Sub(exp, retrieveMetricsKey)
	return retrieveMetrics{
		kvRetrieve: alamos.NewGaugeDuration(
			sub,
			alamos.Debug,
			"kvRetrieveDuration",
		),
		dataRead: alamos.NewGaugeDuration(
			sub,
			alamos.Debug,
			"dataReadDuration",
		),
		segSize: alamos.NewGauge[int](
			sub,
			alamos.Debug,
			"segSize",
		),
		segCount: alamos.NewGauge[int](
			sub,
			alamos.Debug,
			"segCount",
		),
		request: alamos.NewGaugeDuration(
			sub,
			alamos.Debug,
			"requestDur",
		),
	}
}
