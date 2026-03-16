// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ = Describe("Log", func() {
	Describe("NewLogger", func() {
		It("Should correctly attach a new logger to the Instrumentation", func() {
			logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{ZapConfig: zap.NewDevelopmentConfig()}))
			i := alamos.New("test", alamos.WithLogger(logger))
			Expect(i.L).ToNot(BeNil())
		})
	})

	Describe("Basic Logging", func() {

		var (
			logger *alamos.Logger
			buffer *bytes.Buffer
		)

		BeforeEach(func() {
			var err error
			config := zap.NewDevelopmentConfig()
			config.OutputPaths = []string{"stdout"}
			logger, err = alamos.NewLogger(alamos.LoggerConfig{ZapConfig: config})
			Expect(err).ToNot(HaveOccurred())
			buffer = &bytes.Buffer{}
			zapLogger := logger.Zap().WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
				return zapcore.NewCore(
					zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
					zapcore.AddSync(buffer),
					zapcore.DebugLevel,
				)
			}))
			logger, err = alamos.NewLogger(alamos.LoggerConfig{ZapLogger: zapLogger})
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should handle WithOptions", func() {
			newLogger := logger.WithOptions(zap.AddCallerSkip(1))
			Expect(newLogger).ToNot(BeNil())
			Expect(newLogger.Zap()).ToNot(Equal(logger.Zap()))
		})

		It("Should handle WithConfig", func() {
			newConfig := alamos.LoggerConfig{
				ZapConfig: zap.NewDevelopmentConfig(),
			}
			newLogger, err := logger.WithConfig(newConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(newLogger).ToNot(BeNil())
			Expect(newLogger.Zap()).ToNot(Equal(logger.Zap()))
		})

		It("Should log Info messages", func() {
			logger.Info("test info message", zap.String("key", "value"))
			Expect(buffer.String()).To(ContainSubstring("test info message"))
			Expect(buffer.String()).To(ContainSubstring("key"))
			Expect(buffer.String()).To(ContainSubstring("value"))
			Expect(buffer.String()).To(ContainSubstring("INFO"))
		})

		It("Should log Warn messages", func() {
			logger.Warn("test warn message", zap.String("key", "value"))
			Expect(buffer.String()).To(ContainSubstring("test warn message"))
			Expect(buffer.String()).To(ContainSubstring("key"))
			Expect(buffer.String()).To(ContainSubstring("value"))
			Expect(buffer.String()).To(ContainSubstring("WARN"))
		})

		It("Should log Debug messages", func() {
			logger.Debug("test debug message", zap.String("key", "value"))
			Expect(buffer.String()).To(ContainSubstring("test debug message"))
			Expect(buffer.String()).To(ContainSubstring("key"))
			Expect(buffer.String()).To(ContainSubstring("value"))
			Expect(buffer.String()).To(ContainSubstring("DEBUG"))
		})

		It("Should log Error messages", func() {
			err := errors.New("test error")
			logger.Error("test error message", zap.Error(err))
			Expect(buffer.String()).To(ContainSubstring("test error message"))
			Expect(buffer.String()).To(ContainSubstring("test error"))
			Expect(buffer.String()).To(ContainSubstring("ERROR"))
		})

		It("Should log with multiple fields", func() {
			logger.Info("test multi-field message",
				zap.String("str", "value"),
				zap.Int("int", 42),
				zap.Bool("bool", true),
			)
			Expect(buffer.String()).To(ContainSubstring("test multi-field message"))
			Expect(buffer.String()).To(ContainSubstring("str"))
			Expect(buffer.String()).To(ContainSubstring("value"))
			Expect(buffer.String()).To(ContainSubstring("42"))
			Expect(buffer.String()).To(ContainSubstring("true"))
		})

		It("Should create a named logger", func() {
			namedLogger := logger.Named("test-component")
			Expect(namedLogger).ToNot(BeNil())
			Expect(namedLogger).ToNot(Equal(logger))
		})

		It("Should include name in log output", func() {
			namedLogger := logger.Named("test-component")
			namedLogger.Info("test named message")
			Expect(buffer.String()).To(ContainSubstring("test named message"))
			Expect(buffer.String()).To(ContainSubstring("test-component"))
		})

		It("Should allow formatting in Infof", func() {
			logger.Infof("test number %d", 123)
			Expect(buffer.String()).To(ContainSubstring("test number 123"))
		})

		It("Should support nested names", func() {
			namedLogger := logger.Named("parent").Named("child")
			namedLogger.Info("test nested named message")
			Expect(buffer.String()).To(ContainSubstring("test nested named message"))
			Expect(buffer.String()).To(ContainSubstring("parent.child"))
		})

		It("Should handle nil logger in Named", func() {
			var nilLogger *alamos.Logger
			namedLogger := nilLogger.Named("test")
			Expect(namedLogger).To(BeNil())
		})
	})

	Describe("Noop", func() {
		It("Should not panic when calling a method on a nil logger", func() {
			var l *alamos.Logger
			Expect(func() { l.Debug("test") }).ToNot(Panic())
			Expect(func() { l.Debugf("test") }).ToNot(Panic())
			Expect(func() { l.Info("test") }).ToNot(Panic())
			Expect(func() { l.Infof("test") }).ToNot(Panic())
			Expect(func() { l.Warn("test") }).ToNot(Panic())
			Expect(func() { l.Error("test") }).ToNot(Panic())
			Expect(func() { l.Fatal("test") }).ToNot(Panic())
			Expect(func() { l.DPanic("test") }).ToNot(Panic())
		})
	})

	Describe("CustomCore", func() {
		var (
			core    zapcore.Core
			encoder zapcore.Encoder
			buffer  *bytes.Buffer
			writer  zapcore.WriteSyncer
		)

		BeforeEach(func() {
			encoder = zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig())
			buffer = &bytes.Buffer{}
			writer = zapcore.AddSync(buffer)
			core = zapcore.NewCore(encoder, writer, zapcore.DebugLevel)
		})

		It("Should wrap an existing core", func() {
			custom := alamos.CustomZapCore(core)
			Expect(custom.Enabled(zapcore.DebugLevel)).To(BeTrue())
			Expect(custom.Enabled(zapcore.InfoLevel)).To(BeTrue())
			Expect(custom.Enabled(zapcore.ErrorLevel)).To(BeTrue())
		})

		It("Should handle error stack traces", func() {
			custom := alamos.CustomZapCore(core)
			err := errors.New("test error")
			entry := zapcore.Entry{
				Level:   zapcore.ErrorLevel,
				Message: "test",
			}
			fields := []zapcore.Field{zap.Error(err)}
			Expect(custom.Write(entry, fields)).To(Succeed())
			Expect(buffer.String()).To(ContainSubstring("test error"))
			Expect(buffer.String()).To(ContainSubstring("stack trace"))
			Expect(buffer.String()).To(ContainSubstring("log_test"))
		})

		It("Should handle custom caller information", func() {
			custom := alamos.CustomZapCore(core)
			entry := zapcore.Entry{
				Level:   zapcore.InfoLevel,
				Message: "test",
			}
			fields := []zapcore.Field{zap.String("caller", "custom/caller.go:42")}
			Expect(custom.Write(entry, fields)).To(Succeed())
			Expect(buffer.String()).To(ContainSubstring("custom/caller.go"))
			Expect(buffer.String()).To(ContainSubstring("42"))
		})

		It("Should maintain core functionality", func() {
			custom := alamos.CustomZapCore(core)
			Expect(custom.With([]zapcore.Field{zap.String("key", "value")})).ToNot(BeNil())
			Expect(custom.Sync()).To(Succeed())
		})

		It("Should handle Check in customCore", func() {
			custom := alamos.CustomZapCore(core)
			entry := zapcore.Entry{
				Level:   zapcore.InfoLevel,
				Message: "test",
			}
			checked := &zapcore.CheckedEntry{}
			result := custom.Check(entry, checked)
			Expect(result).ToNot(BeNil())
			Expect(result).To(Equal(checked))
		})

		It("Should log with custom core", func() {
			custom := alamos.CustomZapCore(core)
			zapLogger := zap.New(custom)
			newLogger, err := alamos.NewLogger(alamos.LoggerConfig{ZapLogger: zapLogger})
			Expect(err).ToNot(HaveOccurred())
			newLogger.Info("test message", zap.String("key", "value"))
			Expect(buffer.String()).To(ContainSubstring("test message"))
			Expect(buffer.String()).To(ContainSubstring("key"))
			Expect(buffer.String()).To(ContainSubstring("value"))
		})
	})
})
