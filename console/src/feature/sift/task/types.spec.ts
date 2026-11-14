// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Sift } from "@/feature/sift";

const VALID = {
  ...Sift.Task.STREAM_SCHEMAS.config.parse({}),
  device: "dev",
  asset: "Test stand 1",
  channels: [1],
};

const issues = (config: unknown): string[] =>
  Sift.Task.deployStreamConfigZ.safeParse(config).error?.issues.map((i) => i.message) ??
  [];

describe("Sift.Task.deployStreamConfigZ", () => {
  it("should accept a config with a device, an asset, and channels", () => {
    expect(issues(VALID)).toEqual([]);
  });

  it("should accept a config without labels", () => {
    expect(issues({ ...VALID, labels: [] })).toEqual([]);
  });

  it("should require a device", () => {
    expect(issues({ ...VALID, device: "" })).toEqual(["Device is required"]);
  });

  it("should require an asset", () => {
    expect(issues({ ...VALID, asset: "" })).toEqual(["Asset is required"]);
  });

  it("should require at least one channel", () => {
    expect(issues({ ...VALID, channels: [] })).toEqual(["Select at least one channel"]);
  });
});
