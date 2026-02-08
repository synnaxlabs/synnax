// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build driver

package log_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/driver/internal/log"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type errClosedReader struct{}

func (errClosedReader) Read([]byte) (int, error) { return 0, os.ErrClosed }
func (errClosedReader) Close() error             { return nil }

func parseLogEntries(buffer *bytes.Buffer) []map[string]any {
	var entries []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buffer.String()), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		Expect(json.Unmarshal([]byte(line), &entry)).To(Succeed())
		entries = append(entries, entry)
	}
	return entries
}

// pipeAndCollect writes input to a PipeToLogger goroutine, closes the pipe,
// waits for processing to finish, then returns the parsed JSON log entries.
func pipeAndCollect(
	logger *alamos.Logger,
	started chan<- struct{},
	input string,
) []map[string]any {
	r, w := io.Pipe()
	buffer := &bytes.Buffer{}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(buffer),
		zapcore.DebugLevel,
	)
	logger = MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
		ZapLogger: zap.New(core),
	}))
	done := make(chan struct{})
	go func() {
		defer close(done)
		log.PipeToLogger(r, logger, started)
	}()
	MustSucceed(w.Write([]byte(input)))
	Expect(w.Close()).To(Succeed())
	Eventually(done).Should(BeClosed())
	return parseLogEntries(buffer)
}

var _ = Describe("PipeToLogger", func() {
	var (
		logger *alamos.Logger
		buffer *bytes.Buffer
	)

	BeforeEach(func() {
		buffer = &bytes.Buffer{}
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.AddSync(buffer),
			zapcore.DebugLevel,
		)
		logger = MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
	})

	Describe("glog format", func() {
		It("Should parse an info line into level, message, and caller", func() {
			entries := pipeAndCollect(logger, nil,
				"I20260208 14:34:21.789995 0x1fa2cec40 start.cpp:19] starting Synnax Driver v0.50.6\n",
			)
			Expect(entries).To(HaveLen(1))
			Expect(entries[0]).To(And(
				HaveKeyWithValue("L", "INFO"),
				HaveKeyWithValue("M", "starting Synnax Driver v0.50.6"),
				HaveKeyWithValue("caller", "start.cpp:19"),
				HaveKey("T"),
				Not(HaveKey("N")),
			))
		})

		It("Should parse a warning line", func() {
			entries := pipeAndCollect(logger, nil,
				"W20260208 14:34:21.000000 0x1fa2cec40 rack.cpp:50] something is off\n",
			)
			Expect(entries).To(HaveLen(1))
			Expect(entries[0]).To(And(
				HaveKeyWithValue("L", "WARN"),
				HaveKeyWithValue("M", "something is off"),
				HaveKeyWithValue("caller", "rack.cpp:50"),
			))
		})

		It("Should parse an error line", func() {
			entries := pipeAndCollect(logger, nil,
				"E20260208 14:34:21.000000 0x1fa2cec40 rack.cpp:51] something broke\n",
			)
			Expect(entries).To(HaveLen(1))
			Expect(entries[0]).To(And(
				HaveKeyWithValue("L", "ERROR"),
				HaveKeyWithValue("M", "something broke"),
				HaveKeyWithValue("caller", "rack.cpp:51"),
			))
		})

		It("Should parse a fatal line as error level", func() {
			entries := pipeAndCollect(logger, nil,
				"F20260208 14:34:21.000000 0x1fa2cec40 rack.cpp:99] fatal crash\n",
			)
			Expect(entries).To(HaveLen(1))
			Expect(entries[0]).To(And(
				HaveKeyWithValue("L", "ERROR"),
				HaveKeyWithValue("M", "fatal crash"),
				HaveKeyWithValue("caller", "rack.cpp:99"),
			))
		})

		It("Should parse a debug line", func() {
			entries := pipeAndCollect(logger, nil,
				"D20260208 14:34:21.000000 0x1fa2cec40 opc.cpp:12] connecting to server\n",
			)
			Expect(entries).To(HaveLen(1))
			Expect(entries[0]).To(And(
				HaveKeyWithValue("L", "DEBUG"),
				HaveKeyWithValue("M", "connecting to server"),
				HaveKeyWithValue("caller", "opc.cpp:12"),
			))
		})

		It("Should handle continuation lines as info with previous caller", func() {
			entries := pipeAndCollect(logger, nil,
				"I20260208 14:34:21.790466 0x16d327000 rack.cpp:33] real-time capabilities:\n"+
					"  priority scheduling: yes\n"+
					"  deadline scheduling: not supported\n",
			)
			Expect(entries).To(HaveLen(3))
			Expect(entries[0]).To(And(
				HaveKeyWithValue("L", "INFO"),
				HaveKeyWithValue("M", "real-time capabilities:"),
				HaveKeyWithValue("caller", "rack.cpp:33"),
			))
			Expect(entries[1]).To(And(
				HaveKeyWithValue("L", "INFO"),
				HaveKeyWithValue("M", "priority scheduling: yes"),
				HaveKeyWithValue("caller", "rack.cpp:33"),
			))
			Expect(entries[2]).To(And(
				HaveKeyWithValue("L", "INFO"),
				HaveKeyWithValue("M", "deadline scheduling: not supported"),
				HaveKeyWithValue("caller", "rack.cpp:33"),
			))
		})

		It("Should detect started successfully", func() {
			started := make(chan struct{})
			entries := pipeAndCollect(logger, started,
				"I20260208 14:34:22.000000 0x16d327000 rack.cpp:200] started successfully\n",
			)
			Expect(started).To(BeClosed())
			Expect(entries).To(HaveLen(1))
			Expect(entries[0]).To(And(
				HaveKeyWithValue("L", "INFO"),
				HaveKeyWithValue("M", "started successfully"),
			))
		})

		It("Should update caller across consecutive lines", func() {
			entries := pipeAndCollect(logger, nil,
				"I20260208 14:34:21.000000 0x16d327000 rack.cpp:33] first\n"+
					"I20260208 14:34:21.000001 0x16d327000 config.cpp:50] second\n",
			)
			Expect(entries).To(HaveLen(2))
			Expect(entries[0]).To(HaveKeyWithValue("caller", "rack.cpp:33"))
			Expect(entries[1]).To(HaveKeyWithValue("caller", "config.cpp:50"))
		})
	})

	Describe("bracketed format", func() {
		It("Should use second bracket group as named logger", func() {
			entries := pipeAndCollect(logger, nil,
				"I [mock] [main.go] test message\n",
			)
			Expect(entries).To(HaveLen(1))
			Expect(entries[0]).To(And(
				HaveKeyWithValue("L", "INFO"),
				HaveKeyWithValue("M", "test message"),
				HaveKeyWithValue("N", "[main.go"),
				HaveKeyWithValue("caller", "[mock"),
			))
		})

		It("Should detect started successfully", func() {
			started := make(chan struct{})
			entries := pipeAndCollect(logger, started,
				"I [mock] [main.go] started successfully\n",
			)
			Expect(started).To(BeClosed())
			Expect(entries).To(HaveLen(1))
			Expect(entries[0]).To(HaveKeyWithValue("M", "started successfully"))
		})
	})

	It("Should warn on empty lines", func() {
		entries := pipeAndCollect(logger, nil, "\n")
		Expect(entries).To(HaveLen(1))
		Expect(entries[0]).To(And(
			HaveKeyWithValue("L", "WARN"),
			HaveKeyWithValue("M", "received empty log line from driver"),
			Not(HaveKey("caller")),
		))
	})

	It("Should return cleanly when reader closes", func() {
		r, w := io.Pipe()
		done := make(chan struct{})
		go func() {
			defer close(done)
			log.PipeToLogger(r, logger, nil)
		}()
		Expect(w.Close()).To(Succeed())
		Eventually(done).Should(BeClosed())
	})

	It("Should suppress os.ErrClosed error", func() {
		done := make(chan struct{})
		go func() {
			defer close(done)
			log.PipeToLogger(errClosedReader{}, logger, nil)
		}()
		Eventually(done).Should(BeClosed())
		Expect(buffer.String()).ToNot(ContainSubstring("ERROR"))
	})
})
