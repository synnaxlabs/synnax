// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table as clientTable } from "@synnaxlabs/client";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Table } from "@/feature/table";
import { client, project } from "@/feature/table/testutil";
import { Session } from "@/session";
import { renderHookWithConsole, uniqueName } from "@/testutil";

describe("Table.useMosaicDrop", () => {
  it("places the table into the target mosaic node", async () => {
    const t = await client.tables.create(await project(), {
      name: uniqueName("table"),
    });
    const { result, store } = await renderHookWithConsole(() => Table.useMosaicDrop(), {
      client,
    });
    result.current({
      id: clientTable.ontologyID(t.key),
      nodeKey: 2,
      location: "bottom",
    });
    await waitFor(() =>
      expect(Session.Layout.select(store.getState(), t.key)?.name).toBe(t.name),
    );
    const layout = Session.Layout.select(store.getState(), t.key);
    expect(layout?.location).toBe("mosaic");
    expect(layout?.tab).toMatchObject({ mosaicKey: 2, location: "bottom" });
  });
});
