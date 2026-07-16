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

import { drifted } from "@/task/drift";
import { type Payload, statusZ } from "@/task/types.gen";

describe("drifted", () => {
  // Both hashes are server-assigned and opaque here; the client only compares them.
  const DEPLOYED = "2de66015b3bdded8";
  const EDITED = "0000000000000000";
  const newPayload = (overrides: {
    running?: boolean;
    taskHash?: string;
    statusHash?: string;
    statusRack?: number;
    hasStatus?: boolean;
  }): Payload => {
    const key = uuid.create();
    const {
      running = true,
      taskHash = DEPLOYED,
      statusHash = DEPLOYED,
      statusRack = 1,
      hasStatus = true,
    } = overrides;
    return {
      key,
      rack: 1,
      name: "test",
      type: "test",
      config: { rate: 50, port: 8080, host: "localhost" },
      configHash: taskHash,
      internal: false,
      snapshot: false,
      status: hasStatus
        ? statusZ().parse({
            message: "running",
            variant: "success",
            details: { task: key, running, configHash: statusHash, rack: statusRack },
          })
        : undefined,
    };
  };

  it("should not drift when the deployed hash and rack match the task", () => {
    expect(drifted(newPayload({}))).toBe(false);
  });

  it("should drift when the stored hash differs from the deployed hash", () => {
    expect(drifted(newPayload({ taskHash: EDITED }))).toBe(true);
  });

  it("should drift when the task moved racks while running", () => {
    expect(drifted(newPayload({ statusRack: 2 }))).toBe(true);
  });

  it("should never drift when the task is not running", () => {
    expect(drifted(newPayload({ running: false, taskHash: EDITED }))).toBe(false);
  });

  it("should never drift without a status", () => {
    expect(drifted(newPayload({ hasStatus: false }))).toBe(false);
  });
});
