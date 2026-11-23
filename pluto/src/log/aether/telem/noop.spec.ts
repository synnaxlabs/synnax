// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { log, LogFactory, type LogSource } from "@/log/aether";

describe("NoopLogSource", () => {
  it("should return an empty array from value()", () => {
    const factory = new LogFactory(null);
    // A log with no telemetry reads from the no-op source.
    const spec = log.logStateZ.shape.telem.parse(undefined);
    const source = factory.create(spec) as LogSource;
    expect(source.value().data).toEqual([]);
  });
});
