// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server

import (
	"cmp"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// normalizePath returns path with one leading '/'. It keeps a trailing '/', which
// servers can route apart from the path without it.
func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	return "/" + strings.TrimLeft(path, "/")
}

// normalize puts the path of each binding of s in the form normalizePath returns and
// sorts each query by parameter, then value.
func normalize(s *Server) {
	for i := range s.Reads {
		r := &s.Reads[i].Request
		r.Path = normalizePath(r.Path)
		sort.SliceStable(r.Query, func(a, b int) bool {
			return cmp.Or(
				strings.Compare(r.Query[a].Parameter, r.Query[b].Parameter),
				strings.Compare(r.Query[a].Value, r.Query[b].Value),
			) < 0
		})
	}
	for i := range s.Writes {
		r := &s.Writes[i].Request
		r.Path = normalizePath(r.Path)
		sort.SliceStable(r.Query, func(a, b int) bool {
			return cmp.Or(
				strings.Compare(r.Query[a].Parameter, r.Query[b].Parameter),
				strings.Compare(r.Query[a].Value, r.Query[b].Value),
			) < 0
		})
	}
}

type requestKey struct {
	method, path, query string
}

// prepare normalizes and validates s. It refuses a read of the same request, pointer,
// and index as another read, and a write of the same request as another write.
func prepare(s *Server) error {
	normalize(s)
	if err := s.Validate(); err != nil {
		return err
	}
	type readKey struct {
		request requestKey
		pointer string
		index   channel.Key
	}
	v := validate.New("Server")
	reads := make(set.Set[readKey], len(s.Reads))
	for i, r := range s.Reads {
		query := make(url.Values, len(r.Request.Query))
		for _, q := range r.Request.Query {
			query.Add(q.Parameter, q.Value)
		}
		k := readKey{
			request: requestKey{
				method: string(r.Request.Method),
				path:   r.Request.Path,
				query:  query.Encode(),
			},
			pointer: r.Pointer,
			index:   r.Index,
		}
		v.Ternaryf(
			fmt.Sprintf("reads.%d", i),
			reads.Contains(k),
			"duplicates the read of %q from %s %s",
			r.Pointer,
			r.Request.Method,
			r.Request.Path,
		)
		reads.Add(k)
	}
	writes := make(set.Set[requestKey], len(s.Writes))
	for i, w := range s.Writes {
		query := make(url.Values, len(w.Request.Query))
		for _, q := range w.Request.Query {
			query.Add(q.Parameter, q.Value)
		}
		k := requestKey{
			method: string(w.Request.Method),
			path:   w.Request.Path,
			query:  query.Encode(),
		}
		v.Ternaryf(
			fmt.Sprintf("writes.%d", i),
			writes.Contains(k),
			"duplicates the write to %s %s",
			w.Request.Method,
			w.Request.Path,
		)
		writes.Add(k)
	}
	return v.Error()
}
