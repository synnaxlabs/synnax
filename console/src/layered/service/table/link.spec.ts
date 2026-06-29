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

import { Table } from "@/layered/service/table";
import { Layout } from "@/layout";
import { renderLinkHook } from "@/testUtils";

describe("Table.useLink", () => {
  it("should place a table layout for the retrieved table", async () => {
    const key = uuid.create();
    const retrieve = vi.fn(async () => ({ key, name: "Sensor Table" }));
    const client = { tables: { retrieve } } as unknown as Client;
    const { handler, store } = renderLinkHook(Table.useLink);
    await handler({ client, key });
    expect(retrieve).toHaveBeenCalledWith({ key });
    expect(Layout.select(store.getState(), key)?.name).toBe("Sensor Table");
  });
});
