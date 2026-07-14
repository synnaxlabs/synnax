// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { drifted, hashConfig } from "@/task/drift";
import { type Payload, statusZ } from "@/task/types.gen";

describe("hashConfig", () => {
  it("should produce the shared cross-language golden hashes", () => {
    expect(hashConfig({})).toEqual("2e1472b57af294d1");
    expect(hashConfig({ rate: 50, port: 8080, host: "localhost" })).toEqual(
      "2de66015b3bdded8",
    );
  });

  it("should hash camelCase keys as their snake_case wire form", () => {
    expect(hashConfig({ dataSaving: true, sampleRate: 5.5 })).toEqual(
      hashConfig({ data_saving: true, sample_rate: 5.5 }),
    );
  });
});

describe("drifted", () => {
  const config = { rate: 50, port: 8080, host: "localhost" };
  const newPayload = (overrides: {
    running?: boolean;
    configHash?: string;
    statusRack?: number;
    hasStatus?: boolean;
  }): Payload => {
    const key = uuid.create();
    const {
      running = true,
      configHash = hashConfig(config),
      statusRack = 1,
      hasStatus = true,
    } = overrides;
    return {
      key,
      rack: 1,
      name: "test",
      type: "test",
      config,
      internal: false,
      snapshot: false,
      status: hasStatus
        ? statusZ().parse({
            message: "running",
            variant: "success",
            details: { task: key, running, configHash, rack: statusRack },
          })
        : undefined,
    };
  };

  it("should not drift when the deployed hash and rack match the row", () => {
    expect(drifted(newPayload({}))).toBe(false);
  });

  it("should drift when the stored config differs from the deployed hash", () => {
    expect(drifted(newPayload({ configHash: "0000000000000000" }))).toBe(true);
  });

  it("should drift when the task moved racks while running", () => {
    expect(drifted(newPayload({ statusRack: 2 }))).toBe(true);
  });

  it("should never drift when the task is not running", () => {
    expect(
      drifted(newPayload({ running: false, configHash: "0000000000000000" })),
    ).toBe(false);
  });

  it("should never drift without a status", () => {
    expect(drifted(newPayload({ hasStatus: false }))).toBe(false);
  });
});
