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
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
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
	rateWindow         = time.Minute
	defaultSampleCount = 100
	maxSampleCount     = 1000
	// sampleWindows is the most windows, spread back in time, that a sample reads from
	// one partition.
	sampleWindows = 3
	// samplePayloadLimit is the most bytes of a record value that a sample returns.
	samplePayloadLimit = 16 << 10
	// sampleByteLimit is the most bytes of record keys, values, and headers that a
	// sample holds while it reads.
	sampleByteLimit = 16 << 20
)

// formatSchemaRegistry is a Confluent Schema Registry payload: a zero byte, a 4-byte
// schema ID, and the encoded value.
const formatSchemaRegistry driver.Format = "schema_registry"

func formatOf(value []byte) driver.Format {
	if len(value) >= 5 && value[0] == 0 {
		return formatSchemaRegistry
	}
	return driver.FormatOf(value)
}

// browsedTopic is one topic of a cluster, as a browse sees it.
type browsedTopic struct {
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
	// Error says why the browse could not read the offsets or the newest record of a
	// partition of the topic. Empty when it read each partition.
	Error string `json:"error" msgpack:"error"`
}

// browseResult is the data that answers a browse.
type browseResult struct {
	// Topics are the topics of the cluster, other than internal topics, sorted by name.
	Topics []browsedTopic `json:"topics" msgpack:"topics"`
	// Truncated is true when the topics do not all fit in the answer.
	Truncated bool `json:"truncated" msgpack:"truncated"`
}

// browseTopics returns the topics of the cluster of props, other than internal topics,
// sorted by name.
func browseTopics(ctx context.Context, props Properties) ([]browsedTopic, error) {
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
		topics   []browsedTopic
		readable []string
	)
	for name, d := range details {
		if d.internal {
			continue
		}
		t := browsedTopic{Name: name, Partitions: d.partitions, Readable: d.readable()}
		topics = append(topics, t)
		if t.Readable {
			readable = append(readable, name)
		}
	}
	slices.SortFunc(
		topics,
		func(a, b browsedTopic) int { return strings.Compare(a.Name, b.Name) },
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
	var (
		newest []fetchRange
		index  = make(map[string]int, len(topics))
	)
	for i := range topics {
		t := &topics[i]
		index[t.Name] = i
		partitions, err := partitionRanges(t.Name, starts, ends)
		for _, p := range partitions {
			t.Records += p.to - p.from
			after, ok := recent.Lookup(p.topic, p.partition)
			if ok && after.Err == nil {
				t.Rate += float64(p.to-after.Offset) / rateWindow.Seconds()
			} else {
				err = cmp.Or(err, after.Err)
			}
			// In a topic written in transactions, the last offset holds the control
			// record that commits the newest records.
			p.from = max(p.from, p.to-2)
			newest = append(newest, p)
		}
		if err != nil {
			t.Error = err.Error()
		}
		if rc, err := configs.On(t.Name, nil); err == nil {
			t.Compacted = compacted(rc)
		}
	}
	errs, err := fetch(ctx, props, newest, func(r *kgo.Record) {
		t := &topics[index[r.Topic]]
		if ts := telem.NewTimeStamp(r.Timestamp); ts > t.Newest {
			t.Newest, t.Format = ts, formatOf(r.Value)
		}
	})
	if err != nil {
		return nil, err
	}
	for topic, err := range errs {
		t := &topics[index[topic]]
		t.Error = cmp.Or(t.Error, err.Error())
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

// partitionRanges returns the offsets from the start to the end of each partition of
// topic that holds records, sorted by partition. It leaves out a partition that is not
// in both starts and ends, or that has an error in either, and returns the first of
// those errors.
func partitionRanges(
	topic string,
	starts, ends kadm.ListedOffsets,
) ([]fetchRange, error) {
	var (
		ranges []fetchRange
		err    error
	)
	for _, end := range ends[topic] {
		start, ok := starts.Lookup(topic, end.Partition)
		if !ok {
			continue
		}
		if listed := cmp.Or(start.Err, end.Err); listed != nil {
			err = cmp.Or(err, listed)
			continue
		}
		if end.Offset <= start.Offset {
			continue
		}
		ranges = append(ranges, fetchRange{
			topic:     topic,
			partition: end.Partition,
			from:      start.Offset,
			to:        end.Offset,
		})
	}
	slices.SortFunc(ranges, func(a, b fetchRange) int {
		return cmp.Compare(a.partition, b.partition)
	})
	return ranges, err
}

// sampleArgs select the records that a sample reads.
type sampleArgs struct {
	// Device is the key of the cluster device to read.
	Device device.Key `json:"device" msgpack:"device"`
	// Topics are the topics to read.
	Topics []string `json:"topics" msgpack:"topics"`
	// Count is the most records to read from each topic. Zero selects 100, and the most
	// is 1,000.
	Count int `json:"count" msgpack:"count"`
}

// header is one header of a record.
type header struct {
	// Key is the key of the header.
	Key string `json:"key" msgpack:"key"`
	// Value is the value of the header, with invalid UTF-8 replaced.
	Value string `json:"value" msgpack:"value"`
}

// sampledRecord is one record that a sample read.
type sampledRecord struct {
	// Partition is the partition of the record.
	Partition int32 `json:"partition" msgpack:"partition"`
	// Offset is the offset of the record in its partition.
	Offset int64 `json:"offset" msgpack:"offset"`
	// Key is the record key, with invalid UTF-8 replaced.
	Key string `json:"key" msgpack:"key"`
	// Time is the time of the record.
	Time telem.TimeStamp `json:"time" msgpack:"time"`
	// Headers are the headers of the record.
	Headers []header `json:"headers" msgpack:"headers"`
	// Format is the format of the whole record value.
	Format driver.Format `json:"format" msgpack:"format"`
	// Payload is the start of the record value, from at most 16 KiB: text when Format
	// is json or text, and hex digits otherwise.
	Payload string `json:"payload" msgpack:"payload"`
}

// sampledTopic is the sample of one topic.
type sampledTopic struct {
	// Topic is the name of the topic.
	Topic string `json:"topic" msgpack:"topic"`
	// Records are the records read, sorted by partition and offset.
	Records []sampledRecord `json:"records" msgpack:"records"`
	// Shapes are the shapes of the JSON values of the records. The source of a shape is
	// a record key.
	Shapes []xjson.Shape `json:"shapes" msgpack:"shapes"`
	// Error says why the sample could not read the topic, or all of the records that it
	// asked for. Empty when it read them.
	Error string `json:"error" msgpack:"error"`
}

// sampleResult is the data that answers a sample.
type sampleResult struct {
	// Topics are the samples, in the order of the topics of the request.
	Topics []sampledTopic `json:"topics" msgpack:"topics"`
	// Truncated is true when the answer leaves out records that the sample read, or
	// when the sample stopped at 16 MiB of record keys, values, and headers. The shapes
	// come from every record that the sample holds.
	Truncated bool `json:"truncated" msgpack:"truncated"`
}

// sampleTopics reads up to args.Count records of each topic of args, in windows spread
// back in time, and infers the shapes of their values. A topic that cannot be read
// carries the reason and does not stop the others. It holds at most sampleByteLimit
// bytes of records. The answer keeps the records that driver.Fit keeps, and the shapes
// come from every record held.
func sampleTopics(
	ctx context.Context,
	props Properties,
	args sampleArgs,
) (sampleResult, error) {
	if len(args.Topics) == 0 {
		return sampleResult{}, errors.Wrap(
			validate.ErrValidation, "topics: must be non-empty",
		)
	}
	for i, topic := range args.Topics {
		if slices.Index(args.Topics, topic) < i {
			return sampleResult{}, errors.Wrapf(
				validate.ErrValidation, "topics: must not repeat %q", topic,
			)
		}
	}
	props.ApplyDefaults()
	count := int64(cmp.Or(args.Count, defaultSampleCount))
	count = min(max(count, 1), maxSampleCount)
	cl, err := openClient(props)
	if err != nil {
		return sampleResult{}, err
	}
	defer cl.Close()
	adm := kadm.NewClient(cl)
	details, err := describeTopics(ctx, cl, args.Topics)
	if err != nil {
		return sampleResult{}, err
	}
	res := sampleResult{Topics: make([]sampledTopic, len(args.Topics))}
	var readable []string
	for i, name := range args.Topics {
		res.Topics[i] = sampledTopic{Topic: name, Records: []sampledRecord{}}
		switch d, ok := details[name]; {
		case !ok || errors.Is(d.err, kerr.UnknownTopicOrPartition):
			res.Topics[i].Error = "The topic does not exist."
		case errors.Is(d.err, kerr.TopicAuthorizationFailed),
			d.err == nil && !d.readable():
			res.Topics[i].Error = "The cluster does not let the client read the topic."
		case d.err != nil:
			res.Topics[i].Error = d.err.Error()
		default:
			readable = append(readable, name)
		}
	}
	if len(readable) == 0 {
		return res, nil
	}
	starts, ends, err := listOffsets(ctx, adm, readable)
	if err != nil {
		return sampleResult{}, err
	}
	var (
		records    = make(map[string][]*kgo.Record)
		errs       = make(map[string]error)
		partitions = make([][]fetchRange, len(readable))
		held       int
		full       bool
	)
	for i, topic := range readable {
		if partitions[i], err = partitionRanges(topic, starts, ends); err != nil {
			errs[topic] = err
		}
	}
	for _, ranges := range sampleRanges(partitions, count) {
		failed, err := fetch(ctx, props, ranges, func(r *kgo.Record) {
			size := recordSize(r)
			if full || held+size > sampleByteLimit {
				full = true
				return
			}
			held += size
			records[r.Topic] = append(records[r.Topic], r)
		})
		if err != nil {
			return sampleResult{}, err
		}
		for topic, err := range failed {
			if errs[topic] == nil {
				errs[topic] = err
			}
		}
		if full {
			break
		}
	}
	sampled := make([][]sampledRecord, len(res.Topics))
	for i := range res.Topics {
		t := &res.Topics[i]
		if err := errs[t.Topic]; err != nil {
			t.Error = err.Error()
		}
		rs := records[t.Topic]
		slices.SortFunc(rs, func(a, b *kgo.Record) int {
			return cmp.Or(
				cmp.Compare(a.Partition, b.Partition),
				cmp.Compare(a.Offset, b.Offset),
			)
		})
		docs := make([]xjson.Document, len(rs))
		for j, r := range rs {
			rec := newSampledRecord(r)
			sampled[i] = append(sampled[i], rec)
			docs[j] = xjson.Document{Source: rec.Key, Data: r.Value, Time: rec.Time}
		}
		t.Shapes = xjson.Infer(docs)
	}
	kept, truncated, err := driver.Fit(res, sampled)
	if err != nil {
		return sampleResult{}, err
	}
	for i := range res.Topics {
		res.Topics[i].Records = kept[i]
	}
	res.Truncated = truncated || full
	return res, nil
}

// recordSize returns the bytes of the key, value, and headers of r.
func recordSize(r *kgo.Record) int {
	n := len(r.Key) + len(r.Value)
	for _, h := range r.Headers {
		n += len(h.Key) + len(h.Value)
	}
	return n
}

func newSampledRecord(r *kgo.Record) sampledRecord {
	s := sampledRecord{
		Partition: r.Partition,
		Offset:    r.Offset,
		Key:       strings.ToValidUTF8(string(r.Key), "�"),
		Time:      telem.NewTimeStamp(r.Timestamp),
		Headers:   make([]header, len(r.Headers)),
	}
	s.Format = formatOf(r.Value)
	s.Payload = driver.Display(r.Value, s.Format, samplePayloadLimit)
	for i, h := range r.Headers {
		s.Headers[i] = header{
			Key:   h.Key,
			Value: strings.ToValidUTF8(string(h.Value), "�"),
		}
	}
	return s
}

// sampleRanges splits count records of each topic among its partitions, and the records
// of each partition among up to sampleWindows windows spread evenly from its start to
// its end. Each element of topics holds the partitionRanges of one topic. A partition
// gives one range to each window it reads.
func sampleRanges(topics [][]fetchRange, count int64) [][]fetchRange {
	windows := make([][]fetchRange, sampleWindows)
	for _, partitions := range topics {
		n := int64(len(partitions))
		for i, p := range partitions {
			budget := count / n
			if int64(i) < count%n {
				budget++
			}
			for w, r := range partitionWindows(p.from, p.to, budget) {
				r.topic, r.partition = p.topic, p.partition
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
// commits, and calls visit with each record in offset order. It skips control records.
// It returns the first error of each topic, once it has read every range or
// fetchTimeout passes. A topic with a range that it did not read in that time gets an
// error.
func fetch(
	ctx context.Context,
	props Properties,
	ranges []fetchRange,
	visit func(*kgo.Record),
) (map[string]error, error) {
	var (
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
		return errs, nil
	}
	// A range of a topic written in transactions can end on a control record, so the
	// client keeps them to see that the range is read.
	cl, err := openClient(
		props,
		kgo.ConsumePartitions(offsets),
		kgo.FetchMaxWait(fetchMaxWait),
		kgo.KeepControlRecords(),
	)
	if err != nil {
		return nil, err
	}
	defer cl.Close()
	fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	done := func(topic string, partition int32) {
		if _, ok := ends[topic][partition]; ok {
			delete(ends[topic], partition)
			left--
		}
	}
	for left > 0 && fetchCtx.Err() == nil {
		fetches := cl.PollFetches(fetchCtx)
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
			if r.Offset < end && !r.Attrs.IsControl() {
				visit(r)
			}
			if r.Offset >= end-1 {
				done(r.Topic, r.Partition)
			}
		})
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for topic, partitions := range ends {
		if len(partitions) > 0 && errs[topic] == nil {
			errs[topic] = errors.Newf(
				"The cluster did not send every record within %s.",
				fetchTimeout,
			)
		}
	}
	return errs, nil
}
