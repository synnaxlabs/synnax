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

import { Kafka } from "@/feature/kafka";
import { assertDefined } from "@/testutil";

describe("plotChannels", () => {
  it("should find the bound channels of the enabled records", () => {
    const plotChannels = Kafka.Task.INTEGRATION.plotChannels?.[Kafka.Task.READ_TYPE];
    assertDefined(plotChannels, "read tasks have no channels to plot");
    const record = (disabled: boolean, ...channels: number[][]) => ({
      disabled,
      icd: uuid.create(),
      messages: channels.map((keys) => ({
        message: uuid.create(),
        bindings: keys.map((channel) => ({ field: uuid.create(), channel })),
      })),
    });
    const config = { records: [record(false, [1, 0], [2]), record(true, [3])] };
    expect(plotChannels(config)).toEqual([1, 2]);
  });
});
