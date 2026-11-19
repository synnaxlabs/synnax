// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import { createWithHook } from "@/platform/task/testutil";

describe("EtherCAT.Task create", () => {
  it("should keep the channels of the given read config", async () => {
    const config = EtherCAT.Task.READ_SCHEMAS.config.parse({ autoStart: true });
    const tsk = await createWithHook(
      EtherCAT.Task.useCreateRead,
      EtherCAT.Task.READ_SCHEMAS,
      { config },
    );
    expect(tsk.config.autoStart).toBe(true);
    expect(tsk.config.channels).toEqual([]);
  });

  it("should keep the channels of the given write config", async () => {
    const config = EtherCAT.Task.WRITE_SCHEMAS.config.parse({ autoStart: true });
    const tsk = await createWithHook(
      EtherCAT.Task.useCreateWrite,
      EtherCAT.Task.WRITE_SCHEMAS,
      { config },
    );
    expect(tsk.config.autoStart).toBe(true);
    expect(tsk.config.channels).toEqual([]);
  });
});
