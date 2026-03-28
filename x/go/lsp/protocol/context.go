// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package protocol

import (
	"context"

	"go.uber.org/zap"
)

type ctxLoggerKey struct{}
type ctxClientKey struct{}

var (
	ctxLogger ctxLoggerKey
	ctxClient ctxClientKey
)

// WithLogger returns the context with zap.Logger value.
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxLogger, logger)
}

// LoggerFromContext extracts zap.Logger from context.
func LoggerFromContext(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(ctxLogger).(*zap.Logger)
	if !ok {
		return zap.NewNop()
	}

	return logger
}

// WithClient returns the context with Client value.
func WithClient(ctx context.Context, client Client) context.Context {
	return context.WithValue(ctx, ctxClient, client)
}
