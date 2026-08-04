// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package clean

import "context"

type Service struct{ name string }

func (*Service) Type() string { return "channel" }

func (s *Service) Name() string { return s.name }

func Anonymous(context.Context, string) {}

func Mixed(ctx context.Context, _ string) { _ = ctx }

func Used(ctx context.Context) { _ = ctx }

func None() {}
