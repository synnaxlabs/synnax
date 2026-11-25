// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json

import (
	"bytes"
	"cmp"
	"encoding/json/jsontext"
	"io"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

const (
	// maxEnumLabels is the most distinct strings of a field that infers as an enum.
	maxEnumLabels = 16
	// maxEnumLabelLength is the longest string, in bytes, of a field that infers as an
	// enum.
	maxEnumLabelLength = 64
	// maxInferredFields is the most fields that one shape holds.
	maxInferredFields = 1024
	// maxClusters is the most clusters of documents that one source splits into.
	maxClusters = 16
	// maxFieldSets is the most sets of fields of one source that split compares.
	maxFieldSets = 256
	// maxSampleLength is the longest sample, in bytes, of a field of a SampledShape.
	maxSampleLength = 64
)

// Document is one sampled JSON document.
type Document struct {
	// Source is where the document came from, such as a record key or a topic.
	Source string
	// Data is the JSON text of the document.
	Data []byte
	// Time is when the document was written. Zero selects the time of inference.
	Time telem.TimeStamp
}

// InferredField is a scalar field that inference found in a set of documents.
type InferredField struct {
	// Pointer locates the field in a document.
	Pointer Pointer `json:"pointer" msgpack:"pointer"`
	// DataType is the data type of a channel that takes the values of the field.
	DataType telem.DataType `json:"data_type" msgpack:"data_type"`
	// Kind is the kind of a payload field that holds the values of the field.
	Kind payload.Kind `json:"kind" msgpack:"kind"`
	// Type is the JSON type of most values of the field.
	Type Type `json:"type" msgpack:"type"`
	// TimeFormat is the format of the values of a timestamp field. Nil for any other
	// field.
	TimeFormat *TimeFormat `json:"time_format,omitempty" msgpack:"time_format,omitempty"`
	// Enum holds the distinct values, sorted, of a string field that repeats a few.
	Enum []string `json:"enum" msgpack:"enum"`
	// Count is the number of documents that hold a value of the field.
	Count int `json:"count" msgpack:"count"`
	// Mismatched is the number of values that a channel of DataType does not take.
	Mismatched int `json:"mismatched" msgpack:"mismatched"`
}

// Shape is the layout of one cluster of documents, from one or more sources.
type Shape struct {
	// Sources are the sources of the documents, in the order first seen.
	Sources []string `json:"sources" msgpack:"sources"`
	// Fields are the scalar fields of the documents, in the order first seen.
	Fields []InferredField `json:"fields" msgpack:"fields"`
	// Index is the position in Fields of the timestamp that indexes the documents, or
	// -1 when no field can.
	Index int `json:"index" msgpack:"index"`
	// Count is the number of documents of the shape.
	Count int `json:"count" msgpack:"count"`
	// Present is the position in Fields of the field that tells the documents of the
	// shape apart from the other shapes of its sources, or -1 when it has none.
	Present int `json:"present" msgpack:"present"`
}

// SampledShape is a Shape with the last value of each of its fields.
type SampledShape struct {
	Shape
	// Samples holds the start of the JSON text of the last value of each of Fields, in
	// the same order: at most 64 bytes, cut at a rune boundary.
	Samples []string
}

// Infer returns the shapes of docs, which are in the order that each source wrote them.
//   - The documents of a source that hold the same fields are one cluster. A cluster
//     whose fields are all in one widest cluster joins it, so a field that some
//     documents lack does not split a cluster. A source with more than 16 clusters or
//     256 sets of fields is one cluster.
//   - A document with no fields joins the one widest cluster of its source, or no
//     cluster.
//   - The clusters of every source that hold the same fields share a shape.
//   - A shape's present field is its first field, other than the whole document, that
//     every one of its documents holds and that no other shape of its sources holds. A
//     shape that is the only shape of each of its sources has none. The clusters of a
//     source whose shapes have none merge into one, so each source has at most one
//     shape with none.
//   - A field is a JSON pointer to a scalar. Arrays and nulls give no field.
//   - Types widen across documents: int64, then uint64, then float64. A boolean field
//     gives uint8. The JSON type of most values wins, and the rest are mismatched.
//   - A string field that repeats a few values of at most 64 bytes gives an enum.
//   - A field whose name suggests a time and whose values all fit one format near the
//     document time gives a timestamp. Unix units come from the magnitude.
//   - The index is the timestamp nearest the document time that rises in each source.
//
// A document that is not JSON gives no fields, and a source with no JSON document gives
// no shape.
func Infer(docs []Document) []Shape {
	now := telem.Now()
	var (
		sources []*inferSource
		byName  = make(map[string]*inferSource)
	)
	for _, doc := range docs {
		leaves, err := parseLeaves(doc.Data)
		if err != nil {
			continue
		}
		src, ok := byName[doc.Source]
		if !ok {
			src = newInferSource(doc.Source, math.MaxInt)
			byName[doc.Source] = src
			sources = append(sources, src)
		}
		ref := doc.Time
		if ref <= 0 {
			ref = now
		}
		src.add(parsedDocument{leaves: leaves, time: ref})
	}
	inf := infer(sources)
	shapes := make([]Shape, len(inf.groups))
	for i := range shapes {
		shapes[i] = inf.shape(i)
	}
	return shapes
}

// inference is what infer finds in the documents of its sources.
type inference struct {
	groups []*inferGroup
	// present holds the position of the present field of each group, or -1.
	present []int
	// clusters holds the clusters of each source, and of holds the group of each.
	clusters [][]cluster
	of       [][]int
}

// infer returns the groups of the documents of sources.
func infer(sources []*inferSource) inference {
	clusters := make([][]cluster, len(sources))
	for i, src := range sources {
		clusters[i] = src.split()
	}
	for {
		groups, of := group(sources, clusters)
		present := presentFields(groups, of)
		if !mergeUnmarked(sources, clusters, of, present) {
			return inference{
				groups:   groups,
				present:  present,
				clusters: clusters,
				of:       of,
			}
		}
	}
}

// shape returns the shape of the group at i.
func (inf inference) shape(i int) Shape {
	s := inf.groups[i].shape()
	s.Present = inf.present[i]
	return s
}

// groupOf returns the position of the group that takes the documents of the set at k of
// the source at i, or -1 when the set joins no cluster.
func (inf inference) groupOf(i, k int) int {
	for c, cl := range inf.clusters[i] {
		if slices.Contains(cl.sets, k) {
			return inf.of[i][c]
		}
	}
	return -1
}

// group returns the groups of clusters, where the clusters that hold the same fields
// share a group, and of, which holds the group of each cluster of each source.
func group(sources []*inferSource, clusters [][]cluster) ([]*inferGroup, [][]int) {
	var (
		groups []*inferGroup
		byKey  = make(map[string]int)
		of     = make([][]int, len(sources))
	)
	for i, src := range sources {
		of[i] = make([]int, len(clusters[i]))
		for k, c := range clusters[i] {
			g, ok := byKey[c.key]
			if !ok {
				g = len(groups)
				byKey[c.key] = g
				groups = append(groups, newInferGroup())
			}
			groups[g].sources = append(groups[g].sources, src.name)
			groups[g].merge(src, c.sets)
			of[i][k] = g
		}
	}
	return groups, of
}

// presentFields returns the position of the present field of each group, or -1. A
// present field is not at, above, or inside a field of another group of its sources, so
// the present rules of a source never nest.
func presentFields(groups []*inferGroup, of [][]int) []int {
	others := sharers(groups, of)
	above := make([]set.Set[string], len(groups))
	for i, g := range groups {
		above[i] = g.above()
	}
	present := make([]int, len(groups))
	for i, g := range groups {
		present[i] = -1
		if len(others[i]) == 0 {
			continue
		}
		present[i] = g.presentField(func(p Pointer) bool {
			for o := range others[i] {
				if groups[o].nests(p, above[o]) {
					return false
				}
			}
			return true
		})
	}
	return present
}

// sharers returns, for each group, the other groups that share a source with it.
func sharers(groups []*inferGroup, of [][]int) []set.Set[int] {
	others := make([]set.Set[int], len(groups))
	for i := range others {
		others[i] = make(set.Set[int])
	}
	for _, gs := range of {
		for _, g := range gs {
			for _, o := range gs {
				if o != g {
					others[g].Add(o)
				}
			}
		}
	}
	return others
}

// mergeUnmarked merges the clusters of each source that has more than one cluster in a
// group with no present field. It reports whether it merged any.
func mergeUnmarked(
	sources []*inferSource,
	clusters [][]cluster,
	of [][]int,
	present []int,
) bool {
	merged := false
	for i, src := range sources {
		var (
			marked   []cluster
			unmarked []int
			count    int
		)
		for k, c := range clusters[i] {
			if present[of[i][k]] >= 0 {
				marked = append(marked, c)
				continue
			}
			unmarked = append(unmarked, c.sets...)
			count++
		}
		if count < 2 {
			continue
		}
		clusters[i] = append(marked, src.merge(unmarked))
		slices.SortFunc(clusters[i], func(a, b cluster) int {
			return cmp.Compare(a.sets[0], b.sets[0])
		})
		merged = true
	}
	return merged
}

// Inferrer infers the shapes of the documents of one source as they arrive, with the
// rules of Infer. Past 1,024 fields across its sets of fields, the documents are one
// shape. It keeps counts and the start of the last value of each field, not documents,
// so its memory does not grow with the number of documents. It is not safe for
// concurrent use.
type Inferrer struct {
	src *inferSource
	// last is the position in src.sets of the set of the last document added, or -1.
	last int
}

// NewInferrer returns an Inferrer that holds no documents.
func NewInferrer() *Inferrer {
	return &Inferrer{src: newInferSource("", maxInferredFields), last: -1}
}

// Add adds data, written at t. Data that is not one JSON value adds nothing.
func (i *Inferrer) Add(data []byte, t telem.TimeStamp) {
	leaves, err := parseLeaves(data)
	if err != nil {
		return
	}
	i.last = i.src.add(parsedDocument{leaves: leaves, time: t})
}

// Shapes returns the shapes of the documents added, with no sources, and the position
// in them of the shape of the last JSON document added. The position is -1 when no JSON
// document was added, or when the last one joins no shape.
func (i *Inferrer) Shapes() ([]SampledShape, int) {
	inf := infer([]*inferSource{i.src})
	shapes := make([]SampledShape, len(inf.groups))
	for k, g := range inf.groups {
		shapes[k] = SampledShape{Shape: inf.shape(k), Samples: g.samples()}
		shapes[k].Sources = nil
	}
	if i.last < 0 {
		return shapes, -1
	}
	return shapes, inf.groupOf(0, i.last)
}

type inferSource struct {
	name string
	// sets holds each set of fields of the documents, in the order first seen.
	sets  []fieldSet
	byKey map[string]int
	// disordered holds, for each time field, the pairs of sets where a value of one is
	// older than the last value of the other.
	disordered map[string]set.Set[[2]int]
	// held is the number of fields that the sets hold.
	held int
	// limit is the most fields that the sets hold before the source has one set.
	limit int
	// whole is true once the documents have more than 256 sets of fields, or the sets
	// would hold more than limit fields. The one set in sets then takes every document.
	whole bool
	// documents is the number of documents added.
	documents int
}

func newInferSource(name string, limit int) *inferSource {
	return &inferSource{
		name:       name,
		byKey:      make(map[string]int),
		disordered: make(map[string]set.Set[[2]int]),
		limit:      limit,
	}
}

// fieldSet is a set of field pointers in their text form, with the stats of the
// documents that hold exactly those fields.
type fieldSet struct {
	// key is the sorted pointers, joined by NUL.
	key    string
	fields set.Set[string]
	group  *inferGroup
}

func newFieldSet(fields set.Set[string]) fieldSet {
	sorted := fields.Slice()
	slices.Sort(sorted)
	return fieldSet{key: strings.Join(sorted, "\x00"), fields: fields}
}

// cluster is the sets of fields of a source whose documents share a shape.
type cluster struct {
	key string
	// sets are the positions of the sets in inferSource.sets, in order.
	sets []int
}

// add adds doc and returns the position in sets of the set that takes it.
func (src *inferSource) add(doc parsedDocument) int {
	fields := make(set.Set[string], len(doc.leaves))
	for _, l := range doc.leaves {
		fields.Add(l.key)
	}
	k := src.setOf(fields)
	src.documents++
	doc.number = src.documents
	src.sets[k].group.addDocument(src.name, doc)
	for _, l := range doc.leaves {
		src.recordDisorder(k, l.key)
	}
	return k
}

// setOf returns the position in sets of the set that takes the documents that hold
// fields.
func (src *inferSource) setOf(fields set.Set[string]) int {
	if src.whole {
		return 0
	}
	fs := newFieldSet(fields)
	if k, ok := src.byKey[fs.key]; ok {
		return k
	}
	if len(src.sets) == maxFieldSets || src.held+len(fields) > src.limit {
		src.collapse()
		return 0
	}
	src.held += len(fields)
	fs.group = newInferGroup()
	src.byKey[fs.key] = len(src.sets)
	src.sets = append(src.sets, fs)
	return len(src.sets) - 1
}

// collapse merges every set of src into one set that takes every later document.
func (src *inferSource) collapse() {
	g := newInferGroup()
	g.merge(src, src.union()[0].sets)
	src.sets = []fieldSet{{group: g}}
	src.byKey, src.disordered, src.whole = nil, nil, true
}

// recordDisorder records each set whose last value of the time field key is newer than
// the value that the set at k just took.
func (src *inferSource) recordDisorder(k int, key string) {
	s := src.sets[k].group.fields[key]
	if s == nil || !s.timestamp() {
		return
	}
	ts := s.time.last[src.name]
	for o, other := range src.sets {
		theirs := other.group.fields[key]
		if o == k || theirs == nil || !theirs.timestamp() ||
			theirs.time.last[src.name] <= ts {
			continue
		}
		pairs, ok := src.disordered[key]
		if !ok {
			pairs = make(set.Set[[2]int])
			src.disordered[key] = pairs
		}
		pairs.Add([2]int{min(k, o), max(k, o)})
	}
}

// split returns the clusters of src, in the order of their first documents.
func (src *inferSource) split() []cluster {
	if src.whole {
		fields := make(set.Set[string], len(src.sets[0].group.fields))
		for key := range src.sets[0].group.fields {
			fields.Add(key)
		}
		return []cluster{{key: newFieldSet(fields).key, sets: []int{0}}}
	}
	into := src.joins()
	var (
		clusters []cluster
		at       = make(map[int]int)
	)
	for k, target := range into {
		if target < 0 {
			continue
		}
		c, ok := at[target]
		if !ok {
			c = len(clusters)
			at[target] = c
			clusters = append(clusters, cluster{key: src.sets[target].key})
		}
		clusters[c].sets = append(clusters[c].sets, k)
	}
	if len(clusters) > maxClusters {
		return src.union()
	}
	return clusters
}

// union returns one cluster that holds every set of src.
func (src *inferSource) union() []cluster {
	sets := make([]int, len(src.sets))
	for i := range sets {
		sets[i] = i
	}
	return []cluster{src.merge(sets)}
}

// merge returns one cluster that holds the sets of src at sets.
func (src *inferSource) merge(sets []int) cluster {
	slices.Sort(sets)
	fields := make(set.Set[string])
	for _, k := range sets {
		fields.Add(src.sets[k].fields.Slice()...)
	}
	return cluster{key: newFieldSet(fields).key, sets: sets}
}

// joins returns, for each set of fields of src, the set whose cluster its documents
// join: the one widest set that holds all of its fields, or else itself. A set is
// widest when no other set holds all of its fields. An empty set that two widest sets
// hold joins no cluster: -1.
func (src *inferSource) joins() []int {
	holds := func(a, b int) bool {
		return len(src.sets[a].fields) > len(src.sets[b].fields) &&
			src.sets[b].fields.IsSubsetOf(src.sets[a].fields)
	}
	n := len(src.sets)
	widest := make([]bool, n)
	for i := range n {
		widest[i] = true
		for j := range n {
			if holds(j, i) {
				widest[i] = false
				break
			}
		}
	}
	into := make([]int, n)
	for i := range n {
		into[i] = i
		if widest[i] {
			continue
		}
		holders := 0
		for j := range n {
			if widest[j] && holds(j, i) {
				into[i] = j
				holders++
			}
		}
		if holders > 1 {
			into[i] = i
			if len(src.sets[i].fields) == 0 {
				into[i] = -1
			}
		}
	}
	return into
}

type parsedDocument struct {
	leaves []leaf
	time   telem.TimeStamp
	// number is the position of the document in its source, from 1.
	number int
}

// leaf is one scalar of a document.
type leaf struct {
	pointer Pointer
	// key is the text form of pointer.
	key  string
	kind jsontext.Kind
	raw  []byte
}

// parseLeaves returns the scalars of data in document order. It returns an error when
// data is not one JSON value.
func parseLeaves(data []byte) ([]leaf, error) {
	var leaves []leaf
	dec := jsontext.NewDecoder(bytes.NewReader(data))
	if err := walkLeaves(dec, nil, &leaves); err != nil {
		return nil, err
	}
	if _, err := dec.ReadToken(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after the JSON value")
	}
	return leaves, nil
}

func walkLeaves(dec *jsontext.Decoder, pointer Pointer, leaves *[]leaf) error {
	switch dec.PeekKind() {
	case '{':
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		for dec.PeekKind() != '}' {
			name, err := dec.ReadToken()
			if err != nil {
				return err
			}
			child := append(slices.Clip(pointer), name.String())
			if err = walkLeaves(dec, child, leaves); err != nil {
				return err
			}
		}
		_, err := dec.ReadToken()
		return err
	case '"', '0', 't', 'f':
		value, err := dec.ReadValue()
		if err != nil {
			return err
		}
		*leaves = append(*leaves, leaf{
			pointer: pointer,
			key:     pointer.String(),
			kind:    value.Kind(),
			raw:     bytes.Clone(value),
		})
		return nil
	default:
		return dec.SkipValue()
	}
}

type inferGroup struct {
	sources []string
	fields  map[string]*fieldStats
	order   []*fieldStats
	count   int
}

// above returns the pointer of each field of g and of each value above it.
func (g *inferGroup) above() set.Set[string] {
	above := make(set.Set[string])
	for _, s := range g.order {
		for k := 1; k <= len(s.pointer); k++ {
			above.Add(s.pointer[:k].String())
		}
	}
	return above
}

// nests reports whether p is at, above, or inside a field of g. above is g.above().
func (g *inferGroup) nests(p Pointer, above set.Set[string]) bool {
	if above.Contains(p.String()) {
		return true
	}
	for k := 1; k < len(p); k++ {
		if _, ok := g.fields[p[:k].String()]; ok {
			return true
		}
	}
	return false
}

// presentField returns the position of the first field that every document of g holds
// and that apart accepts, or -1. It is a timestamp only when no value field can be.
func (g *inferGroup) presentField(apart func(Pointer) bool) int {
	// A value field names the kind of its documents, and a timestamp does not.
	for _, timestamp := range []bool{false, true} {
		if k := slices.IndexFunc(g.order, func(s *fieldStats) bool {
			// A present rule at the root meets every document.
			return s.timestamp() == timestamp && s.count == g.count &&
				len(s.pointer) > 0 && apart(s.pointer)
		}); k >= 0 {
			return k
		}
	}
	return -1
}

func newInferGroup() *inferGroup {
	return &inferGroup{fields: make(map[string]*fieldStats)}
}

// merge adds the documents of the sets of src at sets. A time field does not rise when
// two of the sets are out of order in it.
func (g *inferGroup) merge(src *inferSource, sets []int) {
	for _, k := range sets {
		from := src.sets[k].group
		g.count += from.count
		for _, o := range from.order {
			key := o.pointer.String()
			s, ok := g.fields[key]
			if !ok {
				if len(g.order) == maxInferredFields {
					continue
				}
				s = newFieldStats(o.pointer)
				g.fields[key] = s
				g.order = append(g.order, s)
			}
			s.merge(o)
		}
	}
	members := set.New(sets...)
	for key, pairs := range src.disordered {
		s, ok := g.fields[key]
		if !ok {
			continue
		}
		for p := range pairs {
			if members.Contains(p[0]) && members.Contains(p[1]) {
				s.time.rising = false
				break
			}
		}
	}
}

func (g *inferGroup) addDocument(source string, doc parsedDocument) {
	g.count++
	for _, l := range doc.leaves {
		s, ok := g.fields[l.key]
		if !ok {
			if len(g.order) == maxInferredFields {
				continue
			}
			s = newFieldStats(l.pointer)
			g.fields[l.key] = s
			g.order = append(g.order, s)
		}
		s.add(source, l, doc.time, doc.number)
	}
}

// samples returns the sample of each field of g, in the order of its fields.
func (g *inferGroup) samples() []string {
	samples := make([]string, len(g.order))
	for i, s := range g.order {
		samples[i] = string(s.sample)
	}
	return samples
}

func (g *inferGroup) shape() Shape {
	shape := Shape{
		Sources: g.sources,
		Fields:  make([]InferredField, len(g.order)),
		Index:   -1,
		Count:   g.count,
		Present: -1,
	}
	nearest := math.Inf(1)
	for i, s := range g.order {
		shape.Fields[i] = s.field()
		// An index must stamp every document.
		if !s.timestamp() || !s.time.rising || s.count < g.count {
			continue
		}
		if d := s.time.distance / float64(s.count); d < nearest {
			shape.Index, nearest = i, d
		}
	}
	return shape
}

// fieldStats accumulates the values of one field.
type fieldStats struct {
	pointer                    Pointer
	count                      int
	numbers, strings, booleans int
	// unreadable is the number of numbers that do not fit a float64.
	unreadable int
	// floating is true when a number has a fraction or does not fit a 64-bit integer.
	floating, unsigned, negative bool
	labels                       set.Set[string]
	// longLabel is true once a string is longer than an enum label can be.
	longLabel bool
	time      timeStats
	// sample is the start of the last value, and sampled is the number of its document.
	sample  []byte
	sampled int
}

// timeStats tracks whether the values of a field are timestamps.
type timeStats struct {
	// candidate is true while every value fits format near its document time.
	candidate bool
	format    TimeFormat
	// formatted is true once a value sets format.
	formatted bool
	// rising is true while no value is older than the value before it in its source.
	rising bool
	last   map[string]telem.TimeStamp
	// distance is the sum of the distances of the values from their document times.
	distance float64
}

// merge adds the values of o. The last value of each source is the newest of the two,
// which is the last value heard when the values rise.
func (t *timeStats) merge(o timeStats) {
	if !t.candidate || !o.candidate || !o.formatted {
		t.candidate = t.candidate && o.candidate
		return
	}
	if t.formatted && t.format != o.format {
		t.candidate = false
		return
	}
	t.format, t.formatted = o.format, true
	t.rising = t.rising && o.rising
	t.distance += o.distance
	for source, ts := range o.last {
		t.last[source] = max(t.last[source], ts)
	}
}

func newFieldStats(pointer Pointer) *fieldStats {
	s := &fieldStats{pointer: pointer, labels: make(set.Set[string])}
	if len(pointer) > 0 && timeName(pointer[len(pointer)-1]) {
		s.time = timeStats{
			candidate: true,
			rising:    true,
			last:      make(map[string]telem.TimeStamp),
		}
	}
	return s
}

// add adds l, the value of the field in document number of source, written at ref.
func (s *fieldStats) add(source string, l leaf, ref telem.TimeStamp, number int) {
	s.count++
	s.sample, s.sampled = sampleOf(l.raw), number
	switch l.kind {
	case '0':
		s.numbers++
		switch n, err := parseNumber(l.raw); {
		case err != nil:
			s.unreadable++
		case n.kind == floatNumber:
			s.floating = true
		case n.kind == uintNumber:
			s.unsigned = true
		case n.i < 0:
			s.negative = true
		}
	case '"':
		s.strings++
		s.addLabel(l.raw)
	default:
		s.booleans++
	}
	s.addTime(source, l, ref)
}

// addLabel keeps the text of raw as a label while the field can still infer as an enum.
func (s *fieldStats) addLabel(raw []byte) {
	if s.longLabel || len(s.labels) > maxEnumLabels {
		return
	}
	text, err := jsontext.AppendUnquote(nil, raw)
	if err != nil {
		return
	}
	if len(text) > maxEnumLabelLength {
		s.longLabel = true
		return
	}
	s.labels.Add(string(text))
}

func (s *fieldStats) addTime(source string, l leaf, ref telem.TimeStamp) {
	t := &s.time
	if !t.candidate {
		return
	}
	ts, format, ok := timeOf(l, ref)
	if !ok || (t.formatted && format != t.format) {
		t.candidate = false
		return
	}
	t.format, t.formatted = format, true
	if last, seen := t.last[source]; seen && ts < last {
		t.rising = false
	}
	t.last[source] = ts
	t.distance += math.Abs(float64(ts) - float64(ref))
}

// merge adds the values of o, which is a field at the same pointer.
func (s *fieldStats) merge(o *fieldStats) {
	s.count += o.count
	s.numbers += o.numbers
	s.strings += o.strings
	s.booleans += o.booleans
	s.unreadable += o.unreadable
	s.floating = s.floating || o.floating
	s.unsigned = s.unsigned || o.unsigned
	s.negative = s.negative || o.negative
	s.longLabel = s.longLabel || o.longLabel
	for label := range o.labels {
		if len(s.labels) > maxEnumLabels {
			break
		}
		s.labels.Add(label)
	}
	s.time.merge(o.time)
	if o.sampled > s.sampled {
		s.sample, s.sampled = o.sample, o.sampled
	}
}

// sampleOf returns the start of raw, at most maxSampleLength bytes, cut at a rune
// boundary. It copies a longer raw, so that the sample does not hold all of it.
func sampleOf(raw []byte) []byte {
	if len(raw) <= maxSampleLength {
		return raw
	}
	n := maxSampleLength
	for n > 0 && !utf8.RuneStart(raw[n]) {
		n--
	}
	return bytes.Clone(raw[:n])
}

func (s *fieldStats) timestamp() bool { return s.time.candidate && s.time.formatted }

func (s *fieldStats) field() InferredField {
	f := InferredField{Pointer: s.pointer, Count: s.count}
	numeric := s.numbers + s.booleans
	switch {
	case s.numbers > 0 && numeric >= s.strings:
		f.Type, f.Mismatched = Number, s.strings+s.unreadable
		f.DataType, f.Kind = s.numberType()
	case s.numbers == 0 && s.booleans > s.strings:
		f.Type, f.DataType, f.Kind = Boolean, telem.Uint8T, payload.KindBoolean
		f.Mismatched = s.strings
	default:
		f.Type, f.DataType, f.Kind, f.Mismatched = String, telem.StringT,
			payload.KindString, numeric
	}
	if s.timestamp() {
		f.DataType, f.Kind = telem.TimestampT, payload.KindTimestamp
		f.TimeFormat, f.Mismatched = new(s.time.format), 0
		return f
	}
	if f.Type == String && !s.longLabel && len(s.labels) <= maxEnumLabels &&
		len(s.labels) < s.strings {
		f.Enum = make([]string, 0, len(s.labels))
		for label := range s.labels {
			f.Enum = append(f.Enum, label)
		}
		slices.Sort(f.Enum)
	}
	return f
}

func (s *fieldStats) numberType() (telem.DataType, payload.Kind) {
	switch {
	case s.floating, s.unsigned && s.negative:
		return telem.Float64T, payload.KindFloat
	case s.unsigned:
		return telem.Uint64T, payload.KindUnsigned
	default:
		return telem.Int64T, payload.KindInteger
	}
}

// timeName reports whether a field name suggests that the field holds a time.
func timeName(name string) bool {
	lower := strings.ToLower(name)
	switch lower {
	case "t", "ts", "epoch":
		return true
	}
	return strings.Contains(lower, "time") ||
		strings.Contains(lower, "date") ||
		strings.Contains(lower, "stamp") ||
		strings.HasSuffix(lower, "_at") ||
		strings.HasSuffix(lower, "_ts") ||
		strings.HasSuffix(name, "At") ||
		strings.HasSuffix(name, "Ts")
}

// unixFormats are the numeric time formats, in rising order of resolution.
var unixFormats = [...]TimeFormat{
	UnixSecond,
	UnixMillisecond,
	UnixMicrosecond,
	UnixNanosecond,
}

// timeOf parses l as a timestamp. A string holds ISO 8601 or a Unix number, and a
// number is Unix in the unit that puts it nearest ref. It reports false when the value
// is not a timestamp within a factor of two of ref.
func timeOf(l leaf, ref telem.TimeStamp) (telem.TimeStamp, TimeFormat, bool) {
	text := l.raw
	switch l.kind {
	case '"':
		unquoted, err := jsontext.AppendUnquote(nil, l.raw)
		if err != nil {
			return 0, 0, false
		}
		if ts, err := parseISO8601(bytes.Clone(unquoted)); err == nil {
			return ts, ISO8601, nearTime(ts, ref)
		}
		text = unquoted
	case '0':
	default:
		return 0, 0, false
	}
	n, err := parseNumber(text)
	if err != nil {
		return 0, 0, false
	}
	format, nearest := UnixSecond, math.Inf(1)
	for _, f := range unixFormats {
		d := math.Abs(n.float()*math.Pow10(f.nanosExponent()) - float64(ref))
		if d < nearest {
			format, nearest = f, d
		}
	}
	ts, err := parseUnix(text, format)
	if err != nil {
		return 0, 0, false
	}
	return ts, format, nearTime(ts, ref)
}

// nearTime reports whether ts is within a factor of two of ref, both measured from the
// Unix epoch.
func nearTime(ts, ref telem.TimeStamp) bool {
	t, r := float64(ts), float64(ref)
	return t >= r/2 && t <= r*2
}
