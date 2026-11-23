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
  it("should find the enabled fields of an endpoint that reads an ICD", () => {
    const plotChannels = HTTP.Task.INTEGRATION.plotChannels?.[HTTP.Task.READ_TYPE];
    assertDefined(plotChannels, "read tasks have no channels to plot");
    const config = {
      endpoints: [
        {
          icd: uuid.create(),
          messages: [
            {
              message: uuid.create(),
              bindings: [{ field: uuid.create(), channel: 1 }],
            },
          ],
          fields: [
            { channel: 2, disabled: false },
            { channel: 3, disabled: true },
          ],
        },
      ],
    };
    expect(plotChannels(config)).toEqual([2]);
  });
});
