// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { type telem } from "@synnaxlabs/x";

/** Converts a line plot's stored custom range to the window the range input edits. */
export const toWindow = (custom: lineplot.CustomRange): telem.Window =>
  custom.variant === "dynamic"
    ? { variant: "dynamic", span: custom.span }
    : { variant: "static", timeRange: { start: custom.start, end: custom.end } };

/** Converts an edited window back to a line plot's stored custom range. */
export const fromWindow = (window: telem.Window): lineplot.CustomRange =>
  window.variant === "dynamic"
    ? { variant: "dynamic", span: window.span }
    : { variant: "static", start: window.timeRange.start, end: window.timeRange.end };
