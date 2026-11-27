// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { assertDefined } from "@synnaxlabs/x/testutil";
import { describe, expect, it } from "vitest";

import { Kafka } from "@/feature/kafka";

describe("plotChannels", () => {
  it("should find the bound channels and fields of the enabled records of every topic", () => {
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
    const read = record(false, [1, 0], [2]);
    const config = {
      topics: [
        { name: "a", records: [read, record(true, [3])] },
        { name: "b", records: [record(false, [4])] },
      ],
    };
    const plotted = plotChannels(config);
    expect(plotted.map(({ channel }) => channel)).toEqual([1, 0, 2, 4]);
    expect(plotted[2].field).toEqual({
      icd: read.icd,
      message: read.messages[1].message,
      key: read.messages[1].bindings[0].field,
    });
  });
});
