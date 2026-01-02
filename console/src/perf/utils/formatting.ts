// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Shared formatting utilities for the performance dashboard.
 * Consolidates all format functions from Dashboard, network, and longtasks modules.
 */

/** Not available constant for null/undefined values */
export const NA = "N/A";

/** No data constant (em dash) for missing values */
export const NO_DATA = "—";

export const formatTime = (seconds: number): string => {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
};

export const formatPercent = (value: number | null): string =>
  value != null ? `${value.toFixed(1)}%` : NA;

export const formatMB = (value: number | null): string =>
  value != null ? `${value.toFixed(1)} MB` : NA;

export const formatFps = (value: number | null): string =>
  value != null ? value.toFixed(1) : NA;

export const formatCount = (value: number | null): string =>
  value != null ? value.toString() : NA;

export const formatDuration = (durationMs: number): string => {
  if (durationMs < 1000) return `${durationMs.toFixed(1)} ms`;
  return `${(durationMs / 1000).toFixed(1)} s`;
};

export const formatAge = (ageMs: number): string => {
  if (ageMs < 1000) return `${Math.floor(ageMs)}ms ago`;
  if (ageMs < 60000) return `${Math.floor(ageMs / 1000)}s ago`;
  return `${Math.floor(ageMs / 60000)}m ago`;
};

export const formatPair = (
  first: number | null,
  second: number | null,
  suffix = "",
): string => {
  if (first == null && second == null) return NO_DATA;
  const firstStr = first != null ? first.toFixed(1) : NO_DATA;
  const secondStr = second != null ? second.toFixed(1) : NO_DATA;
  return `${firstStr} / ${secondStr}${suffix}`;
};

export const formatDelta = (
  start: number | null,
  end: number | null,
  suffix = "",
): string => {
  if (start == null || end == null) return NO_DATA;
  const delta = end - start;
  const sign = delta >= 0 ? "+" : "";
  return `${sign}${delta.toFixed(1)}${suffix}`;
};

export const formatPercentChange = (
  percent: number | null,
  invertSign = false,
): string => {
  if (percent == null) return NO_DATA;
  const value = invertSign ? -percent : percent;
  const sign = value >= 0 ? "+" : "";
  return `${sign}${value.toFixed(1)}%`;
};

export const truncateEndpoint = (endpoint: string, segments = 2): string => {
  const parts = endpoint.split("/").filter((p) => p.length > 0);
  if (parts.length <= segments) return endpoint;
  return `/${parts.slice(-segments).join("/")}`;
};
