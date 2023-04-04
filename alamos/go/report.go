// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos

import (
	"bytes"
	"encoding/json"
	"go.uber.org/zap"
	"io"
)

type Reporter interface {
	Report() Report
}

type Report map[string]interface{}

type ReportManager struct {
}

func AttachReporter(i Instrumentation, key string, report Reporter) {
	ins, ok := Extract(ctx)
	if !ok {
		return
	}
	ins.attachReporter(key, report)
}

// JSON writes the report as JSON as bytes.
func (r Report) JSON() ([]byte, error) {
	b := bytes.NewBuffer([]byte{})
	err := r.WriteJSON(b)
	return b.Bytes(), err
}

// WriteJSON writes the report as JSON to the given writer.
func (r Report) WriteJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "")
	return e.Encode(r)
}

func (r Report) String() string {
	b, err := r.JSON()
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func (r Report) ZapFields() []zap.Field {
	args := make([]zap.Field, 0, len(r))
	for k, v := range r {
		args = append(args, zap.Any(k, v))
	}
	return args
}
