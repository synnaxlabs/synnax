// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@/telem";

/** A wait that expires once, after a span. */
export interface Timer {
  /** Resolves once the span passes. Never resolves after cancel. */
  expired: Promise<void>;
  /** Stops the wait and releases its resources. Safe to call more than once. */
  cancel: () => void;
}

/** Starts a {@link Timer} that expires after the span. */
export type Start = (span: CrudeTimeSpan) => Timer;

/** Starts a {@link Timer} on the system clock. */
export const start: Start = (span) => {
  const { promise, resolve } = Promise.withResolvers<void>();
  const handle = setTimeout(resolve, new TimeSpan(span).milliseconds);
  return { expired: promise, cancel: () => clearTimeout(handle) };
};
