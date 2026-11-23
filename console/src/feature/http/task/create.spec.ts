// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it } from "vitest";

import { HTTP } from "@/feature/http";
import { createTestDevice } from "@/platform/device/testutil";
import { createWithHook } from "@/platform/task/testutil";

const client = createTestClient();

describe("HTTP.Task create", () => {
  it("should create a read task on the given device", async () => {
    const dev = await createTestDevice(client);
    const tsk = await createWithHook(HTTP.Task.useCreateRead, HTTP.Task.READ_SCHEMAS, {
      deviceKey: dev.key,
    });
    expect(tsk.config.device).toBe(dev.key);
  });

  it("should create a write task on the given device", async () => {
    const dev = await createTestDevice(client);
    const tsk = await createWithHook(
      HTTP.Task.useCreateWrite,
      HTTP.Task.WRITE_SCHEMAS,
      {
        deviceKey: dev.key,
      },
    );
    expect(tsk.config.device).toBe(dev.key);
  });
});
