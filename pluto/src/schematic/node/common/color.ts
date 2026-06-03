// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";

import { Theming } from "@/theming";

/// resolveColor returns the theme's default symbol color when c is unset or the ZERO
/// "follow-theme" sentinel, and otherwise honors the explicit color.
export const resolveColor = (
  c: color.Crude | undefined,
  theme: Theming.Theme,
): color.Crude => (c == null || color.isZero(c) ? theme.colors.gray.l11 : c);

/// useColor resolves a symbol's configured color against the active theme; primitives
/// that already hold a theme should call resolveColor directly.
export const useColor = (c?: color.Crude): color.Crude =>
  resolveColor(c, Theming.use());
