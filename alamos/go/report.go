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
	"github.com/synnaxlabs/x/config"
	"go.uber.org/zap"
	"io"
)

type ReportProvider interface {
	Report() Report
}

type ReporterConfig struct {
}

var (
	_                     config.Config[ReporterConfig] = ReporterConfig{}
	DefaultReporterConfig                               = ReporterConfig{}
)

func (r ReporterConfig) Validate() error {
	return nil
}

func (r ReporterConfig) Override(other ReporterConfig) ReporterConfig {
	return r
}

type Reporter struct {
	meta    InstrumentationMeta
	reports map[string]ReportProvider
}

var _ sub[*Reporter] = (*Reporter)(nil)

func NewReporter(configs ...ReporterConfig) (*Reporter, error) {
	return &Reporter{}, nil
}

func (r *Reporter) Attach(key string, report ReportProvider, level Level) {
	if r == nil {
		return
	}
	if r.reports == nil {
		r.reports = make(map[string]ReportProvider)
	}
	r.reports[key] = report
}

func (r *Reporter) sub(meta InstrumentationMeta) *Reporter {
	if r == nil {
		return nil
	}
	return &Reporter{meta: meta, reports: r.reports}
}

type Report map[string]interface{}

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

func (r Report) ZapFields() []zap.Field { return r.zapFields("") }

func (r Report) zapFields(prefix string) []zap.Field {
	args := make([]zap.Field, 0, len(r))
	for k, v := range r {
		if v_, ok := v.(Report); ok {
			args = append(args, v_.zapFields(prefix+k+".")...)
			continue
		}
		args = append(args, zap.Any(prefix+k, v))
	}
	return args
}
