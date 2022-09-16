package allocate

import (
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"math"
	"sync"
)

// Allocator is used to allocate items to a set of descriptors. A descriptor can represent a file, buffer (essentially
// anything assigned with a maximum size).
//
// Type Arguments:
//
// K - The type of the items key.
// D - The type of the descriptors key.
//
// Implementation Details:
//
// The implementation of Allocator provided in this package optimizes for:
// sequential allocation of items with the same key (i.e. all items with the key "hello" go to their own file
// (if possible)). It uses a simple set of rules:
//
//  1. Allocate the item to the descriptor it was allocated to previously.
//  2. If the item is not allocated OR the size of the descriptor exceeds the size set in Config.MaxSize,
//     then allocate the item to the next AVAILABLE descriptor.
//
// Available Means:
//
//	A. A completely NEW descriptor if config.MaxDescriptors has not been reached.
//	OR
//	B. The descriptor with the lowest size if config.MaxDescriptors has been reached.
type Allocator[K, D comparable, I Item[K]] interface {
	// Allocate allocates itemDescriptors of a given size to descriptors. Returns a slice of descriptor keys.
	Allocate(items ...I) []D
}

func New[K, D comparable, I Item[K]](nd NextDescriptor[D], config Config) Allocator[K, D, I] {
	mergedCfg := mergeConfig(config)
	metrics := newMetrics(mergedCfg.Experiment)
	return &defaultAlloc[K, D, I]{
		config:          mergedCfg,
		descriptorSizes: make(map[D]telem.Size),
		itemDescriptors: make(map[K]D),
		nextD:           nd,
		metrics:         metrics,
	}
}

type Item[K comparable] interface {
	// Key returns the key of the item.
	Key() K
	// Size returns the size of the item.
	Size() telem.Size
}

// NextDescriptor returns a unique descriptor key that represents the next descriptor.
type NextDescriptor[D comparable] interface {
	Next() D
}

const (
	// DefaultMaxDescriptors is the default maximum number of descriptors.
	DefaultMaxDescriptors = 50
	// DefaultMaxSize is the default maximum size of a descriptor in bytes.
	DefaultMaxSize = 5e8
)

type Config struct {
	// MaxDescriptors is the maximum number of concurrent descriptors that can be allocated at once.
	// If this value is 0, the default value of DefaultMaxDescriptors is used.
	MaxDescriptors int
	// MaxSize is the maximum size of a descriptor in bytes. If this value is 0, the default value of
	// DefaultMaxSize is used.
	MaxSize telem.Size
	// Experiment is the experiment that Allocate will use to record its metrics.
	Experiment alamos.Experiment
}

var _ config.Config[Config] = Config{}

func (cfg Config) Override(other Config) Config {
	cfg.MaxSize = override.Numeric(cfg.MaxSize, other.MaxSize)
	cfg.MaxDescriptors = override.Numeric(cfg.MaxDescriptors, other.MaxDescriptors)
	cfg.Experiment = override.Nil(cfg.Experiment, other.Experiment)
	return cfg
}

func (cfg Config) Validate() error {
	v := validate.New("cesium.allocate")
	validate.Positive(v, "MaxDescriptors", cfg.MaxDescriptors)
	validate.Positive(v, "MaxSize", cfg.MaxSize)
	return v.Error()
}

var DefaultConfig = Config{
	MaxDescriptors: 50,
	MaxSize:        5e8,
}

func mergeConfig(c Config) Config {
	if c.MaxDescriptors == 0 {
		c.MaxDescriptors = DefaultMaxDescriptors
	}
	if c.MaxSize == 0 {
		c.MaxSize = DefaultMaxSize
	}
	return c
}

type defaultAlloc[K, D comparable, I Item[K]] struct {
	mu              sync.Mutex
	descriptorSizes map[D]telem.Size
	itemDescriptors map[K]D
	nextD           NextDescriptor[D]
	config          Config
	metrics         Metrics
}

// Allocate implements the Allocator interface.
func (d *defaultAlloc[K, D, I]) Allocate(items ...I) []D {
	d.mu.Lock()
	defer d.mu.Unlock()
	dKeys := make([]D, len(items))
	for i, item := range items {
		dKeys[i] = d.allocate(item)
	}
	return dKeys
}

func (d *defaultAlloc[K, D, I]) allocate(item I) D {
	sw := d.metrics.Allocate.Stopwatch()
	sw.Start()
	defer sw.Stop()
	// By default, allocate to the same descriptor as the previous item.
	key, ok := d.itemDescriptors[item.Key()]
	// If we can't find the item, allocated it to a new descriptor.
	if !ok {
		key = d.new(item)
	}
	size, ok := d.descriptorSizes[key]
	if !ok {
		panic("[cesium.allocate] - descriptor not found")
	}
	// If the descriptor is too large, allocate a new descriptor.
	if (size + item.Size()) > d.config.MaxSize {
		key = d.new(item)
	}
	d.descriptorSizes[key] += item.Size()
	return key
}

func (d *defaultAlloc[K, D, I]) new(item I) (key D) {
	// Remove any descriptors that are too large.
	d.scrubOversized()
	// If we've reached our limit, allocate to the descriptor with the smallest size.
	if len(d.descriptorSizes) >= d.config.MaxDescriptors {
		key = d.smallestDescriptor()
	} else {
		// If we haven't reached our limit, allocate to a new descriptor.
		key = d.newDescriptor()
	}
	d.itemDescriptors[item.Key()] = key
	return key
}

func (d *defaultAlloc[K, D, I]) newDescriptor() D {
	n := d.nextD.Next()
	d.descriptorSizes[n] = 0
	return n
}

func (d *defaultAlloc[K, D, I]) scrubOversized() {
	for key, size := range d.descriptorSizes {
		if size > d.config.MaxSize {
			delete(d.descriptorSizes, key)
		}
	}
}

func (d *defaultAlloc[K, D, I]) smallestDescriptor() (desc D) {
	min := telem.Size(math.MaxInt)
	for k, size := range d.descriptorSizes {
		if size < min {
			desc = k
		}
	}
	return desc
}

type NextDescriptorInt struct {
	v int
}

func (n *NextDescriptorInt) Next() int { n.v++; return n.v }
