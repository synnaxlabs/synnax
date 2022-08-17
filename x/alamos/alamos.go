package alamos

// |||||| EXPERIMENT ||||||

// Experiment is alamos' core data type. It represents a hierarchical collection of application metrics.
// Experiment is a tree-like structure where each node is either a metric or a Sub-experiment.
//
// Creating Experiments:
//
// SinkTarget create an experiment, use alamos.New().
//
// Metrics:
//
// SinkTarget add a metric, use one of the metric constructors. Available metrics are:
//
// 		- alamos.NewGauge
//		- alamos.NewSeries
//		- alamos.NewGaugeDuration
//		- alamos.NewSeriesDuration
//
// Each metric in an Experiment is uniquely identified by a string key. The experiment-key combination is used to
// identify the metric in generated reports.
//
// Empty Experiments:
//
// Alamos is designed to be used alongside production code. This means that it is possible to pass nil Experiments
// throughout an application. If a Metric is registered with an empty (nil) Experiment, all of its behavior will appear
// to remain the same, but the Metric will not allocate any memory or record any values. For example:
//
// 		var exp alamos.Experiment
//	    // This gauge will appear to behave normally, but will not allocate memory or record values.
//		g := exp.NewGauge(exp, "bar")
// 		g.Record(1)
//
// The same principle applies for Sub-experiments. If a parent Experiment is empty and Sub is called, the returned
// Sub-experiment will be empty as well.
//
// When approaching empty experiments, we considered taking a route similar to zap.NewNop(), but because alamos
// makes extensive use of generics, and methods can't have type parameters, we decided to try tolerating nil
// experiments instead.
//
// Organizing Experiments:
//
// Only one top-level experiment should be created per application. Sub-experiments should be created to separate
// individual application concerns.
//
type Experiment interface {
	// Key returns the key of the experiment.
	Key() string
	// Report returns a report of all the experiment's metrics.
	Report() Report
	filterTest(level Level) bool
	sub(string) Experiment
	getMetric(string) baseMetric
	addMetric(metric baseMetric)
	attachReporter(string, Level, Reporter)
}

// New creates a new experiment with the given key.
func New(key string, opts ...Option) Experiment {
	o := newOptions(opts...)
	return &experiment{
		key:          key,
		children:     make(map[string]Experiment),
		measurements: make(map[string]baseMetric),
		reporters:    make(map[string]Reporter),
		options:      o,
	}
}

// Sub creates a new Sub-experiment with the given name and adds it to the given experiment.
// If exp is nil, the new Sub-experiment is NOT created, and instead the function returns nil.
func Sub(exp Experiment, key string) Experiment {
	if exp == nil {
		return nil
	}
	return exp.sub(key)
}

func RetrieveMetric[T any](exp Experiment, key string) Metric[T] {
	if exp == nil {
		return nil
	}
	return exp.getMetric(key).(Metric[T])
}

type experiment struct {
	options      *options
	key          string
	children     map[string]Experiment
	measurements map[string]baseMetric
	reports      map[string]Report
	reporters    map[string]Reporter
}

func (e *experiment) Key() string {
	return e.key
}

func (e *experiment) sub(key string) Experiment {
	exp := New(key)
	e.addSub(key, exp)
	return exp
}

func (e *experiment) getMetric(key string) baseMetric {
	return e.measurements[key]
}

func (e *experiment) addMetric(m baseMetric) {
	e.measurements[m.Key()] = m
}

func (e *experiment) addSub(key string, exp Experiment) Experiment {
	e.children[key] = exp
	return exp
}

func (e *experiment) filterTest(level Level) bool {
	for _, filter := range e.options.filters {
		if filter.Test(level) {
			return true
		}
	}
	return false
}
