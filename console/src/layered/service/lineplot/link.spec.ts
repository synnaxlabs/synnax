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

import { LinePlot } from "@/layered/service/lineplot";
import { Layout } from "@/layout";
import { renderLinkHook } from "@/testUtils";

describe("LinePlot.useLink", () => {
  it("should place a line plot layout for the retrieved line plot", async () => {
    const key = uuid.create();
    const retrieve = vi.fn(async () => ({ key, name: "Tank Pressure" }));
    const client = { lineplots: { retrieve } } as unknown as Client;
    const { handler, store } = renderLinkHook(LinePlot.useLink);
    await handler({ client, key });
    expect(retrieve).toHaveBeenCalledWith({ key });
    expect(Layout.select(store.getState(), key)?.name).toBe("Tank Pressure");
  });
});
