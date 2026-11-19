// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"encoding/xml"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

const dataRegionIDAttr = "data-region-id"

type element struct {
	tag     string
	id      string
	classes []string
	// regionID is the data-region-id of the element, or empty.
	regionID string
	parent   *element
	// nth is the position of the element among its siblings with the same tag, from 1.
	nth int
	// start is the byte offset of the element's start tag in the markup.
	start int
}

// tree holds the shapes of an SVG drawing below its root, and gives a data-region-id to
// each shape that a part selects by its position.
type tree struct {
	markup   string
	elements []*element
	taken    set.Set[string]
	added    map[int]string
	next     int
}

// parseTree parses markup as XML. ok is false when the markup does not parse, as a
// browser then draws nothing.
func parseTree(markup string) (t *tree, ok bool) {
	t = &tree{markup: markup, taken: set.Set[string]{}, added: map[int]string{}}
	dec := xml.NewDecoder(strings.NewReader(markup))
	var (
		stack []*element
		// counts[i] maps a tag to how many children of stack[i] have that tag.
		counts []map[string]int
	)
	for {
		start := int(dec.InputOffset())
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return t, len(stack) == 0
		}
		if err != nil {
			return nil, false
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			el := &element{tag: tok.Name.Local, start: start}
			for _, a := range tok.Attr {
				switch {
				case a.Name.Space == "" && a.Name.Local == "id":
					el.id = a.Value
				case a.Name.Space == "" && a.Name.Local == "class":
					el.classes = strings.Fields(a.Value)
				case a.Name.Space == "" && a.Name.Local == dataRegionIDAttr:
					el.regionID = a.Value
					t.taken.Add(a.Value)
				}
			}
			if len(stack) > 0 {
				counts[len(counts)-1][el.tag]++
				el.nth = counts[len(counts)-1][el.tag]
				if len(stack) > 1 {
					el.parent = stack[len(stack)-1]
				}
				t.elements = append(t.elements, el)
			}
			stack = append(stack, el)
			counts = append(counts, map[string]int{})
		case xml.EndElement:
			stack = stack[:len(stack)-1]
			counts = counts[:len(counts)-1]
		}
	}
}

// selectAll returns the shapes below the root that chain selects, in document order.
func (t *tree) selectAll(chain []compound) []*element {
	var out []*element
	for _, el := range t.elements {
		if matchChain(el, chain) {
			out = append(out, el)
		}
	}
	return out
}

func matchChain(el *element, chain []compound) bool {
	for _, c := range slices.Backward(chain) {
		if el == nil || !matchCompound(el, c) {
			return false
		}
		el = el.parent
	}
	return true
}

func matchCompound(el *element, c compound) bool {
	switch {
	case c.id != "":
		return el.id == c.id
	case c.dataRegionID != "":
		return el.regionID == c.dataRegionID
	case len(c.classes) > 0:
		for _, class := range c.classes {
			if !slices.Contains(el.classes, class) {
				return false
			}
		}
		return true
	default:
		return el.tag == c.tag && (c.nth == 0 || el.nth == c.nth)
	}
}

// name returns a selector that names el, giving el a new data-region-id when it has
// none.
func (t *tree) name(el *element) string {
	if el.regionID == "" {
		for {
			t.next++
			id := "region-" + strconv.Itoa(t.next)
			if !t.taken.Contains(id) {
				t.taken.Add(id)
				el.regionID = id
				t.added[el.start] = id
				break
			}
		}
	}
	return `[data-region-id="` + cssEscape(el.regionID) + `"]`
}

// markupWithIDs returns the markup with each new data-region-id written into its start
// tag.
func (t *tree) markupWithIDs() string {
	if len(t.added) == 0 {
		return t.markup
	}
	starts := make([]int, 0, len(t.added))
	for s := range t.added {
		starts = append(starts, s)
	}
	slices.Sort(starts)
	var (
		b    strings.Builder
		prev int
	)
	for _, s := range starts {
		end := s + 1
		for end < len(t.markup) && !isNameEnd(t.markup[end]) {
			end++
		}
		b.WriteString(t.markup[prev:end])
		b.WriteString(` ` + dataRegionIDAttr + `="` + t.added[s] + `"`)
		prev = end
	}
	b.WriteString(t.markup[prev:])
	return b.String()
}

func isNameEnd(b byte) bool { return strings.IndexByte(" \t\r\n/>", b) >= 0 }
