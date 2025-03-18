// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { writeConfigZ } from "./types";

describe("OPC Write Task Types", () => {
  it("should validate the write config", () => {
    const config: z.input<typeof writeConfigZ> = {
      channels: [
        {
          channel: 432,
          cmdChannel: 0,
          dataType: "float",
          enabled: true,
          key: "432",
          name: "test",
          nodeId: "1",
          nodeName: "test",
        },
      ],
      dataSaving: true,
      device: "1",
    };
    const result = writeConfigZ.safeParse(config);
    expect(result.success).toBe(true);
    expect(result.data?.channels[0].cmdChannel).toBe(432);
  });
});
