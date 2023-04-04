// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos

type Instrumentation interface {
	key() string
	l() *Logger
	t() *Tracer
	r() *ReportManager
}

func Dev(key string, silent bool, serviceName string) *Instrumentation {
	return &Instrumentation{
		Key: key,
		L:   newDevLogger(key),
		T:   newDevTracer(serviceName),
	}
}
