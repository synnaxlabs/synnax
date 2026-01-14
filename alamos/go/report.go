// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos

import (
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ReportProvider can provide a Report to a Reporter.
type ReportProvider interface {
	// Report generates and returns a Report.
	Report() Report
}

// ReporterConfig is the configuration for a Reporter.
type ReporterConfig struct {
	// Filter is a function called to determine whether a report should be included
	// for the given key and environment. If the filter returns false, the report
	// will not be included.
	Filter EnvironmentFilter
}

var (
	_ config.Config[ReporterConfig] = ReporterConfig{}
	// DefaultReporterConfig is the default configuration for a Reporter.
	DefaultReporterConfig = ReporterConfig{}
)

// Validate implements config.Config.
func (r ReporterConfig) Validate() error {
	v := validate.New("alamos.reporter_config")
	validate.NotNil(v, "filter", r.Filter)
	return v.Error()
}

// Override implement config.Properties.
func (r ReporterConfig) Override(other ReporterConfig) ReporterConfig {
	r.Filter = override.Nil(r.Filter, other.Filter)
	return r
}

// Reporter is used to attach reports (key-value metadata) to Instrumentation. It's
// typically used for recording the configuration of a service.
type Reporter struct {
	meta    InstrumentationMeta
	reports map[string]ReportProvider
	config  ReporterConfig
}

// NewReporter instantiates a new Reporter using the given configurations. If no configurations
// are provided, the function will return an error. To use a no-op reporter, simply
// pass a nil-pointer.
func NewReporter(configs ...ReporterConfig) (*Reporter, error) {
	cfg, err := config.New(DefaultReporterConfig, configs...)
	if err != nil {
		return nil, err
	}
	return &Reporter{config: cfg}, nil
}

// Debug attaches the given ReportProvider to the Reporter with the given key in the
// Debug environment . The Report is lazily evaluated, and will only be called
// when the instrumentation report is generated.
func (r *Reporter) Debug(key string, report ReportProvider) {
	r.Attach(key, report, EnvironmentDebug)
}

// Prod attaches the given ReportProvider to the Reporter with the given key in the
// production (Prod) environment . The Report is lazily evaluated, and will only be called
// when the instrumentation report is generated.
func (r *Reporter) Prod(key string, report ReportProvider) {
	r.Attach(key, report, EnvironmentProd)
}

// Bench attaches the given ReportProvider to the Reporter with the given key in the
// benchmark (Bench) environment . The Report is lazily evaluated, and will only be called
// when the instrumentation report is generated.
func (r *Reporter) Bench(key string, report ReportProvider) {
	r.Attach(key, report, EnvironmentBench)
}

// Attach attaches the given ReportProvider to the Reporter with the given key under
// the Environment env. The Report is lazily evaluated, and will only be called
// when the instrumentation report is generated.
func (r *Reporter) Attach(key string, report ReportProvider, env Environment) {
	if r == nil {
		return
	}
	if r.reports == nil {
		r.reports = make(map[string]ReportProvider)
	}
	r.reports[key] = report
}

// Report is key-value Metadata that can be attached to an Instrumentation. All values
// stores in a report must be JSON-serializable. Otherwise, it's up to the user to
// decide what to do store. We recommend using snake_case for keys.
type Report map[string]any

// ZapFields generates a set of zap.Fields that can be used to log the report.
func (r Report) ZapFields() []zap.Field { return r.zapFields("") }

func (r Report) zapFields(prefix string) []zap.Field {
	args := make([]zap.Field, 0, len(r))
	for k, v := range r {
		if v, ok := v.(Report); ok {
			args = append(args, v.zapFields(prefix+k+".")...)
			continue
		}
		args = append(args, zap.Any(prefix+k, v))
	}
	return args
}

func (r *Reporter) sub(meta InstrumentationMeta) *Reporter {
	if r == nil {
		return nil
	}
	return &Reporter{meta: meta, reports: r.reports}
}
