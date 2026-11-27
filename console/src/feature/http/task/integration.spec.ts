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

import { HTTP } from "@/feature/http";

describe("plotChannels", () => {
  it("should find the bound channels and fields of the messages of each enabled endpoint", () => {
    const plotChannels = HTTP.Task.INTEGRATION.plotChannels?.[HTTP.Task.READ_TYPE];
    assertDefined(plotChannels, "read tasks have no channels to plot");
    const read = (...messages: number[][]) => ({
      icd: uuid.create(),
      messages: messages.map((channels) => ({
        message: uuid.create(),
        bindings: channels.map((channel) => ({ field: uuid.create(), channel })),
      })),
    });
    const first = read([1, 0], [4]);
    const config = {
      endpoints: [
        { ...first, fields: [{ channel: 2, disabled: false }] },
        { ...read([5]), disabled: true },
        {},
      ],
    };
    const plotted = plotChannels(config);
    expect(plotted.map(({ channel }) => channel)).toEqual([1, 0, 4]);
    expect(plotted[2].field).toEqual({
      icd: first.icd,
      message: first.messages[1].message,
      key: first.messages[1].bindings[0].field,
    });
  });
});
