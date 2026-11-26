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
import { describe, expect, it, vi } from "vitest";

import { HTTP } from "@/feature/http";
import type { FLAGS } from "@/flags";

vi.mock("@/flags/flags", async (importOriginal) => ({
  ...(await importOriginal<typeof FLAGS>()),
  icds: false,
}));

describe("plotChannels with the ICD flag off", () => {
  it("should find the bound channels of the messages of each enabled endpoint", () => {
    const plotChannels = HTTP.Task.INTEGRATION.plotChannels?.[HTTP.Task.READ_TYPE];
    assertDefined(plotChannels, "read tasks have no channels to plot");
    const read = (...messages: number[][]) => ({
      icd: uuid.create(),
      messages: messages.map((channels) => ({
        message: uuid.create(),
        bindings: channels.map((channel) => ({ field: uuid.create(), channel })),
      })),
    });
    const config = {
      endpoints: [
        { ...read([1, 0], [4]), fields: [{ channel: 2, disabled: false }] },
        { ...read([5]), disabled: true },
      ],
    };
    expect(plotChannels(config)).toEqual([1, 4]);
  });
});
