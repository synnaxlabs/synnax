// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package logutil

import "go.uber.org/zap"

// DebugError returns a zap field that can be used to log an error whose presence
// is not exceptional i.e. it does not deserve a stack trace. zap.Error has no way
// to disable stack traces in debug logging, so we use this instead. DebugError should
// only be used in debug logging, and NOT for production errors that are exceptional.
func DebugError(err error) zap.Field {
	if err == nil {
		return zap.Skip()
	}
	return zap.String("error", err.Error())
}
