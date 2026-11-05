// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

const (
	// fetchTimeout bounds one fetch of a browse or a sample.
	fetchTimeout = 3 * time.Second
	// fetchMaxWait is the longest that a broker holds a fetch with no new records.
	fetchMaxWait = 250 * time.Millisecond
	// rateWindow is the time over which a browse measures the rate of a topic.
	rateWindow = time.Minute
	// browsePartitionLimit is the most partitions whose newest record a browse reads.
	browsePartitionLimit = 1000
	defaultSampleCount   = 100
	maxSampleCount       = 1000
	// sampleWindows is the most windows, spread back in time, that a sample reads from
	// one partition.
	sampleWindows = 3
	// samplePayloadLimit is the most bytes of a record value that a sample returns.
	samplePayloadLimit = 16 << 10
)

// Topic is one topic of a cluster, as a browse sees it.
type Topic struct {
	// Name is the name of the topic.
	Name string `json:"name" msgpack:"name"`
	// Partitions is the number of partitions of the topic.
	Partitions int `json:"partitions" msgpack:"partitions"`
	// Records is the number of offsets between the start and the end of each partition.
	// A compacted topic holds fewer records.
	Records int64 `json:"records" msgpack:"records"`
	// Compacted is true when the cleanup policy of the topic compacts it.
	Compacted bool `json:"compacted" msgpack:"compacted"`
	// Rate is the number of records per second written in the last minute.
	Rate float64 `json:"rate" msgpack:"rate"`
	// Newest is the time of the newest record. Zero when the topic holds none.
	Newest telem.TimeStamp `json:"newest" msgpack:"newest"`
	// Format is the format of the newest record. Empty when the topic holds none.
	Format driver.Format `json:"format,omitempty" msgpack:"format,omitempty"`
	// Readable is false when the cluster does not let the client read the topic.
	Readable bool `json:"readable" msgpack:"readable"`
}

// Browse returns the topics of the cluster of props, other than internal topics, sorted
// by name.
func Browse(ctx context.Context, props Properties) ([]Topic, error) {
	props.ApplyDefaults()
	cl, err := openClient(props)
	if err != nil {
		return nil, err
	}
	defer cl.Close()
	adm := kadm.NewClient(cl)
	details, err := describeTopics(ctx, cl, nil)
	if err != nil {
		return nil, err
	}
	var (
		topics   []Topic
		readable []string
	)
	for name, d := range details {
		if d.internal {
			continue
		}
		t := Topic{Name: name, Partitions: d.partitions, Readable: d.readable()}
		topics = append(topics, t)
		if t.Readable {
			readable = append(readable, name)
		}
	}
	slices.SortFunc(
		topics,
		func(a, b Topic) int { return strings.Compare(a.Name, b.Name) },
	)
	if len(readable) == 0 {
		return topics, nil
	}
	starts, ends, err := listOffsets(ctx, adm, readable)
	if err != nil {
		return nil, err
	}
	recent, err := adm.ListOffsetsAfterMilli(
		ctx, time.Now().Add(-rateWindow).UnixMilli(), readable...,
	)
	if err != nil {
		return nil, err
	}
	// A client that may not describe the configs of a topic cannot tell whether it is
	// compacted.
	configs, err := adm.DescribeTopicConfigs(ctx, readable...)
	if authErr := (*kadm.AuthError)(nil); errors.As(err, &authErr) {
		configs = nil
	} else if err != nil {
		return nil, err
	}
	var newest []fetchRange
	for i := range topics {
		t := &topics[i]
		for _, end := range ends[t.Name] {
			start, _ := starts.Lookup(end.Topic, end.Partition)
			after, ok := recent.Lookup(end.Topic, end.Partition)
			if !ok {
				after = end
			}
			t.Records += end.Offset - start.Offset
			t.Rate += float64(end.Offset-after.Offset) / rateWindow.Seconds()
			if end.Offset > start.Offset && len(newest) < browsePartitionLimit {
				newest = append(newest, fetchRange{
					topic:     end.Topic,
					partition: end.Partition,
					from:      end.Offset - 1,
					to:        end.Offset,
				})
			}
		}
		if rc, err := configs.On(t.Name, nil); err == nil {
			t.Compacted = compacted(rc)
		}
	}
	records, _, err := fetch(ctx, props, newest)
	if err != nil {
		return nil, err
	}
	for i := range topics {
		t := &topics[i]
		for _, r := range records[t.Name] {
			if ts := telem.NewTimeStamp(r.Timestamp); ts > t.Newest {
				t.Newest, t.Format = ts, driver.FormatOf(r.Value)
			}
		}
	}
	return topics, nil
}

// topicDetail is what the metadata of a cluster says about one topic.
type topicDetail struct {
	partitions int
	internal   bool
	// operations are the operations that the client may perform on the topic. Empty
	// when the cluster omits them.
	operations []kadm.ACLOperation
	err        error
}

// readable reports whether the client may read the topic. A cluster that omits the
// authorized operations allows each read that it does not deny.
func (d topicDetail) readable() bool {
	if d.err != nil {
		return false
	}
	return len(d.operations) == 0 ||
		slices.Contains(d.operations, kadm.OpRead) ||
		slices.Contains(d.operations, kadm.OpAll)
}

// describeTopics returns the metadata of topics, or of every topic that the client may
// describe when topics is nil. Unlike kadm, it keeps the error of each topic apart, so
// that a denied topic does not fail the others.
func describeTopics(
	ctx context.Context,
	cl *kgo.Client,
	topics []string,
) (map[string]topicDetail, error) {
	req := kmsg.NewPtrMetadataRequest()
	req.IncludeTopicAuthorizedOperations = true
	for _, name := range topics {
		t := kmsg.NewMetadataRequestTopic()
		t.Topic = new(name)
		req.Topics = append(req.Topics, t)
	}
	res, err := req.RequestWith(ctx, cl)
	if err != nil {
		return nil, err
	}
	details := make(map[string]topicDetail, len(res.Topics))
	for _, t := range res.Topics {
		if t.Topic == nil {
			continue
		}
		details[*t.Topic] = topicDetail{
			partitions: len(t.Partitions),
			internal:   t.IsInternal,
			operations: kadm.DecodeACLOperations(t.AuthorizedOperations),
			err:        kerr.ErrorForCode(t.ErrorCode),
		}
	}
	return details, nil
}

func compacted(rc kadm.ResourceConfig) bool {
	for _, c := range rc.Configs {
		if c.Key == "cleanup.policy" && c.Value != nil {
			return strings.Contains(*c.Value, "compact")
		}
	}
	return false
}

func listOffsets(
	ctx context.Context,
	adm *kadm.Client,
	topics []string,
) (starts, ends kadm.ListedOffsets, err error) {
	if starts, err = adm.ListStartOffsets(ctx, topics...); err != nil {
		return nil, nil, err
	}
	if ends, err = adm.ListEndOffsets(ctx, topics...); err != nil {
		return nil, nil, err
	}
	return starts, ends, nil
}

// SampleRequest selects the records that Sample reads.
type SampleRequest struct {
	// Topics are the topics to read.
	Topics []string `json:"topics" msgpack:"topics"`
	// Count is the most records to read from each topic. Zero selects 100, and the most
	// is 1,000.
	Count int `json:"count" msgpack:"count"`
}

// Header is one header of a record.
type Header struct {
	// Key is the key of the header.
	Key string `json:"key" msgpack:"key"`
	// Value is the value of the header, with invalid UTF-8 replaced.
	Value string `json:"value" msgpack:"value"`
}

// SampledRecord is one record that a sample read.
type SampledRecord struct {
	// Partition is the partition of the record.
	Partition int32 `json:"partition" msgpack:"partition"`
	// Offset is the offset of the record in its partition.
	Offset int64 `json:"offset" msgpack:"offset"`
	// Key is the record key, with invalid UTF-8 replaced.
	Key string `json:"key" msgpack:"key"`
	// Time is the time of the record.
	Time telem.TimeStamp `json:"time" msgpack:"time"`
	// Headers are the headers of the record.
	Headers []Header `json:"headers" msgpack:"headers"`
	// Payload is the start of the record value, at most 16 KiB.
	Payload []byte `json:"payload" msgpack:"payload"`
}

// SampledTopic is the sample of one topic.
type SampledTopic struct {
	// Topic is the name of the topic.
	Topic string `json:"topic" msgpack:"topic"`
	// Records are the records read, sorted by partition and offset.
	Records []SampledRecord `json:"records" msgpack:"records"`
	// Shapes are the shapes of the JSON values of the records. The source of a shape is
	// a record key.
	Shapes []xjson.Shape `json:"shapes" msgpack:"shapes"`
	// Error says why the topic could not be read. Empty when it was read.
	Error string `json:"error" msgpack:"error"`
}

// Sample reads up to req.Count records of each topic of req, in windows spread back in
// time, and infers the shapes of their values. A topic that cannot be read carries the
// reason and does not stop the others.
func Sample(
	ctx context.Context,
	props Properties,
	req SampleRequest,
) ([]SampledTopic, error) {
	props.ApplyDefaults()
	count := int64(cmp.Or(req.Count, defaultSampleCount))
	count = min(max(count, 1), maxSampleCount)
	cl, err := openClient(props)
	if err != nil {
		return nil, err
	}
	defer cl.Close()
	adm := kadm.NewClient(cl)
	details, err := describeTopics(ctx, cl, req.Topics)
	if err != nil {
		return nil, err
	}
	samples := make([]SampledTopic, len(req.Topics))
	var readable []string
	for i, name := range req.Topics {
		samples[i] = SampledTopic{Topic: name, Records: []SampledRecord{}}
		switch d, ok := details[name]; {
		case !ok || errors.Is(d.err, kerr.UnknownTopicOrPartition):
			samples[i].Error = "The topic does not exist."
		case errors.Is(d.err, kerr.TopicAuthorizationFailed),
			d.err == nil && !d.readable():
			samples[i].Error = "The cluster does not let the client read the topic."
		case d.err != nil:
			samples[i].Error = d.err.Error()
		default:
			readable = append(readable, name)
		}
	}
	if len(readable) == 0 {
		return samples, nil
	}
	starts, ends, err := listOffsets(ctx, adm, readable)
	if err != nil {
		return nil, err
	}
	var (
		records = make(map[string][]*kgo.Record)
		errs    = make(map[string]error)
	)
	for _, ranges := range sampleRanges(readable, starts, ends, count) {
		got, failed, err := fetch(ctx, props, ranges)
		if err != nil {
			return nil, err
		}
		for topic, rs := range got {
			records[topic] = append(records[topic], rs...)
		}
		for topic, err := range failed {
			if errs[topic] == nil {
				errs[topic] = err
			}
		}
	}
	for i := range samples {
		s := &samples[i]
		if err := errs[s.Topic]; err != nil {
			s.Error = err.Error()
		}
		rs := records[s.Topic]
		slices.SortFunc(rs, func(a, b *kgo.Record) int {
			return cmp.Or(
				cmp.Compare(a.Partition, b.Partition),
				cmp.Compare(a.Offset, b.Offset),
			)
		})
		docs := make([]xjson.Document, len(rs))
		for j, r := range rs {
			s.Records = append(s.Records, sampledRecord(r))
			docs[j] = xjson.Document{
				Source: s.Records[j].Key,
				Data:   r.Value,
				Time:   s.Records[j].Time,
			}
		}
		s.Shapes = xjson.Infer(docs)
	}
	return samples, nil
}

func sampledRecord(r *kgo.Record) SampledRecord {
	s := SampledRecord{
		Partition: r.Partition,
		Offset:    r.Offset,
		Key:       strings.ToValidUTF8(string(r.Key), "�"),
		Time:      telem.NewTimeStamp(r.Timestamp),
		Headers:   make([]Header, len(r.Headers)),
		Payload:   r.Value[:min(len(r.Value), samplePayloadLimit)],
	}
	for i, h := range r.Headers {
		s.Headers[i] = Header{
			Key:   h.Key,
			Value: strings.ToValidUTF8(string(h.Value), "�"),
		}
	}
	return s
}

// sampleRanges splits count records of each topic among its partitions, and the records
// of each partition among up to sampleWindows windows spread evenly from its start to
// its end. A partition gives one range to each window it reads.
func sampleRanges(
	topics []string,
	starts, ends kadm.ListedOffsets,
	count int64,
) [][]fetchRange {
	windows := make([][]fetchRange, sampleWindows)
	for _, topic := range topics {
		var partitions []kadm.ListedOffset
		for _, end := range ends[topic] {
			if start, _ := starts.Lookup(
				topic,
				end.Partition,
			); end.Offset > start.Offset {
				partitions = append(partitions, end)
			}
		}
		slices.SortFunc(partitions, func(a, b kadm.ListedOffset) int {
			return cmp.Compare(a.Partition, b.Partition)
		})
		n := int64(len(partitions))
		for i, end := range partitions {
			budget := count / n
			if int64(i) < count%n {
				budget++
			}
			start, _ := starts.Lookup(end.Topic, end.Partition)
			for w, r := range partitionWindows(start.Offset, end.Offset, budget) {
				r.topic, r.partition = end.Topic, end.Partition
				windows[w] = append(windows[w], r)
			}
		}
	}
	return windows
}

// partitionWindows splits budget records of the offsets from start to end into up to
// sampleWindows ranges, spread evenly, the last ending at end. Their sizes differ by at
// most one and sum to budget.
func partitionWindows(start, end, budget int64) []fetchRange {
	if budget <= 0 {
		return nil
	}
	if end-start <= budget {
		return []fetchRange{{from: start, to: end}}
	}
	var (
		n      = min(sampleWindows, budget)
		gap    = end - start - budget
		ranges = make([]fetchRange, n)
		read   int64
	)
	for k := range n {
		size := budget / n
		if k < budget%n {
			size++
		}
		from := end - size
		if n > 1 {
			from = start + read + gap*k/(n-1)
		}
		ranges[k] = fetchRange{from: from, to: from + size}
		read += size
	}
	return ranges
}

// fetchRange is the offsets from `from` up to `to` of one partition.
type fetchRange struct {
	topic     string
	partition int32
	from, to  int64
}

// fetch reads the records of ranges with one consumer, with no consumer group and no
// commits. It returns the records of each topic and the first error of each topic. It
// returns what it read when fetchTimeout passes.
func fetch(
	ctx context.Context,
	props Properties,
	ranges []fetchRange,
) (map[string][]*kgo.Record, map[string]error, error) {
	var (
		records = make(map[string][]*kgo.Record)
		errs    = make(map[string]error)
		offsets = make(map[string]map[int32]kgo.Offset)
		ends    = make(map[string]map[int32]int64)
		left    int
	)
	for _, r := range ranges {
		if r.from >= r.to {
			continue
		}
		if offsets[r.topic] == nil {
			offsets[r.topic] = make(map[int32]kgo.Offset)
			ends[r.topic] = make(map[int32]int64)
		}
		offsets[r.topic][r.partition] = kgo.NewOffset().At(r.from)
		ends[r.topic][r.partition] = r.to
		left++
	}
	if left == 0 {
		return records, errs, nil
	}
	cl, err := openClient(
		props,
		kgo.ConsumePartitions(offsets),
		kgo.FetchMaxWait(fetchMaxWait),
	)
	if err != nil {
		return nil, nil, err
	}
	defer cl.Close()
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	done := func(topic string, partition int32) {
		if _, ok := ends[topic][partition]; ok {
			delete(ends[topic], partition)
			left--
		}
	}
	for left > 0 && ctx.Err() == nil {
		fetches := cl.PollFetches(ctx)
		fetches.EachError(func(topic string, partition int32, err error) {
			if errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded) {
				return
			}
			if errs[topic] == nil {
				errs[topic] = err
			}
			done(topic, partition)
		})
		fetches.EachRecord(func(r *kgo.Record) {
			end, ok := ends[r.Topic][r.Partition]
			if !ok {
				return
			}
			if r.Offset < end {
				records[r.Topic] = append(records[r.Topic], r)
			}
			if r.Offset >= end-1 {
				done(r.Topic, r.Partition)
			}
		})
	}
	return records, errs, nil
}
