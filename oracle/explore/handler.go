// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package explore

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"time"
)

//go:embed index.html
var page []byte

// Loader analyzes the schemas and returns their model.
type Loader func(ctx context.Context) (Model, error)

// Handler serves the explorer page at / and the model at /model.json. It calls load on
// every model request, so a page reload picks up schema edits.
func Handler(load Loader) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(page))
	})
	mux.HandleFunc("GET /model.json", func(w http.ResponseWriter, r *http.Request) {
		m, err := load(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		b, err := json.Marshal(m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeContent(w, r, "model.json", time.Time{}, bytes.NewReader(b))
	})
	return mux
}
