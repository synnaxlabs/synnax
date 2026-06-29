// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { ChannelServices } from "@/channel/services";
import { Layout } from "@/layout";
import { Project } from "@/project";
import { Range } from "@/range";
import { renderLinkHook } from "@/testUtils";

describe("ChannelServices.useLink", () => {
  it("should create and place a line plot for the retrieved channel", async () => {
    const plotKey = uuid.create();
    const retrieve = vi.fn(async () => ({ key: 65_537, name: "Pressure" }));
    const create = vi.fn(async () => ({ key: plotKey, name: "Pressure Plot" }));
    const client = {
      channels: { retrieve },
      lineplots: { create },
    } as unknown as Client;
    const { handler, store } = renderLinkHook(ChannelServices.useLink, {
      [Project.SLICE_NAME]: Project.reducer,
      [Range.SLICE_NAME]: Range.reducer,
    });
    await handler({ client, key: "65537" });
    expect(retrieve).toHaveBeenCalledWith("65537");
    expect(create).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ channels: { y1: [65_537] } }),
    );
    expect(Layout.select(store.getState(), plotKey)?.name).toBe("Pressure Plot");
  });
});
