// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor

import (
	"context"
	"encoding/binary"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/service/channel/verification"
	tasklegacy "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

const (
	// clusterStateKey holds the persisted Aspen cluster state.
	clusterStateKey = "aspen.cluster"
	// channelCounterSuffix ends the key of a node's leased channel key counter.
	channelCounterSuffix = ".distribution.channel.leasedCounter"
	// freeChannelCounterSuffix ends the key of the bootstrapper's free channel key
	// counter.
	freeChannelCounterSuffix = ".distribution.channel.counter.free"
	// rackCounterSuffix ends the key of a node's rack key counter.
	rackCounterSuffix = ".rack.counter"
)

// bucket names for keys the structural walk recognizes but no table owns.
const (
	bucketMigration = "gorp.migration"
	bucketDigest    = "aspen.digest"
	bucketVersion   = "aspen.version"
	bucketCluster   = "aspen.cluster"
	bucketCounter   = "counter"
	bucketStaging   = "task-staging"
	bucketLicense   = "license"
	bucketUnknown   = "unknown"
)

// Bucket summarizes one class of keys in the store.
type Bucket struct {
	// Name is the table or key class the bucket holds.
	Name string `json:"name"`
	// Entries is the number of keys in the bucket.
	Entries int64 `json:"entries"`
	// Bytes is the total size of the bucket's keys and values.
	Bytes int64 `json:"bytes"`
}

// NodeInfo describes one member of the cluster the store belongs to.
type NodeInfo struct {
	// Key is the node's cluster-unique key.
	Key aspen.NodeKey `json:"key"`
	// Address is the node's advertised address.
	Address string `json:"address"`
	// State is the node's last gossiped membership state.
	State string `json:"state"`
	// Heartbeat is the node's last gossiped heartbeat, as generation.version.
	Heartbeat string `json:"heartbeat"`
}

// ClusterInfo describes the cluster identity persisted in the store.
type ClusterInfo struct {
	// Key is the cluster's unique key. Zero when no state is persisted.
	Key uuid.UUID `json:"key"`
	// HostKey is the key of the node that owns this directory.
	HostKey aspen.NodeKey `json:"host_key"`
	// Nodes holds every node the host knew about, ordered by key.
	Nodes []NodeInfo `json:"nodes"`
}

// KVReport summarizes the contents of the key-value store.
type KVReport struct {
	// Cluster is the cluster identity persisted in the store.
	Cluster ClusterInfo `json:"cluster"`
	// Buckets holds per-class counts, ordered by descending byte total.
	Buckets []Bucket `json:"buckets"`
	// Entries is the total number of keys in the store.
	Entries int64 `json:"entries"`
	// Bytes is the total size of every key and value in the store.
	Bytes int64 `json:"bytes"`
}

// nodeStates names every Aspen membership state.
var nodeStates = map[aspen.NodeState]string{
	aspen.NodeStateHealthy: "healthy",
	aspen.NodeStateSuspect: "suspect",
	aspen.NodeStateDead:    "dead",
	aspen.NodeStateLeft:    "left",
}

// readCluster reads the persisted cluster identity. A store holding no state returns a
// zero ClusterInfo.
func readCluster(ctx context.Context, db kv.DB) (ClusterInfo, error) {
	state, err := aspen.PeekClusterState(ctx, db)
	if err != nil {
		return ClusterInfo{}, errors.Skip(err, query.ErrNotFound)
	}
	info := ClusterInfo{Key: state.ClusterKey, HostKey: state.HostKey}
	for _, n := range state.Nodes {
		info.Nodes = append(info.Nodes, NodeInfo{
			Key:     n.Key,
			Address: string(n.Address),
			State:   nodeStates[n.State],
			Heartbeat: strconv.FormatUint(uint64(n.Heartbeat.Generation), 10) + "." +
				strconv.FormatUint(uint64(n.Heartbeat.Version), 10),
		})
	}
	slices.SortFunc(info.Nodes, func(a, b NodeInfo) int {
		return int(a.Key) - int(b.Key)
	})
	return info, nil
}

// walkResult holds what the structural walk observed beyond the bucket totals.
type walkResult struct {
	// buckets counts entries and bytes per key class.
	buckets map[string]*Bucket
	// counters holds the value of every key counter found, by key.
	counters map[string]int64
	// staging is the number of leftover task-migration staging entries.
	staging int64
	// unknown aggregates keys that belong to no known class.
	unknown violation
	// entries is the total number of keys walked.
	entries int64
	// bytes is the total size of every key and value walked.
	bytes int64
}

// walk buckets every key in the store against the given table names. Values are sized,
// never decoded.
func walk(db kv.DB, tables []string) (res walkResult, err error) {
	res = walkResult{
		buckets:  make(map[string]*Bucket),
		counters: make(map[string]int64),
	}
	// Longest name first, so a key never buckets into a shorter name its table name
	// starts with.
	names := slices.Clone(tables)
	slices.SortFunc(names, func(a, b string) int { return len(b) - len(a) })
	it, err := db.OpenIterator(kv.IteratorOptions{})
	if err != nil {
		return res, err
	}
	defer func() { err = errors.Combine(err, it.Close()) }()
	for it.First(); it.Valid(); it.Next() {
		var (
			key    = string(it.Key())
			size   = int64(len(it.Key()) + len(it.Value()))
			bucket = classify(key, names)
		)
		res.entries++
		res.bytes += size
		b, found := res.buckets[bucket]
		if !found {
			b = &Bucket{Name: bucket}
			res.buckets[bucket] = b
		}
		b.Entries++
		b.Bytes += size
		switch bucket {
		case bucketCounter:
			res.counters[key] = decodeCounter(it.Value())
		case bucketStaging:
			res.staging++
		case bucketUnknown:
			res.unknown.note(key)
		}
	}
	return res, errors.Combine(it.Error(), err)
}

// classify names the bucket a key belongs to. names must be sorted longest first.
func classify(key string, names []string) string {
	if strings.HasPrefix(key, gorp.MigrationKeyPrefix) {
		return bucketMigration
	}
	if after, found := strings.CutPrefix(key, gorp.KeyPrefix); found {
		for _, name := range names {
			if strings.HasPrefix(after, name) {
				return name
			}
		}
		return bucketUnknown
	}
	switch {
	case strings.HasPrefix(key, aspen.DigestPrefix):
		return bucketDigest
	case key == aspen.VersionCounterKey:
		return bucketVersion
	case key == clusterStateKey:
		return bucketCluster
	case strings.HasPrefix(key, tasklegacy.LegacyKeyKVPrefix):
		return bucketStaging
	case key == string(verification.RetrieveKey):
		return bucketLicense
	case isCounter(key):
		return bucketCounter
	}
	return bucketUnknown
}

// isCounter reports whether key names one of the Core's key counters.
func isCounter(key string) bool {
	return strings.HasSuffix(key, channelCounterSuffix) ||
		strings.HasSuffix(key, freeChannelCounterSuffix) ||
		strings.HasSuffix(key, rackCounterSuffix)
}

// decodeCounter reads a counter value. A short value decodes to zero, which the counter
// check then reports as behind the keys the tables hold.
func decodeCounter(b []byte) int64 {
	if len(b) < 8 {
		return 0
	}
	return int64(binary.LittleEndian.Uint64(b))
}
