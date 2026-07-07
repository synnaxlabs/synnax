// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, log } from "@synnaxlabs/client";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Log } from "@/feature/log";
import { Session } from "@/session";
import { renderHookWithConsole, uniqueName } from "@/testutil";

const client = createTestClient();

const createLog = async () => {
  const proj = await client.projects.create({
    name: uniqueName("project"),
    layout: {},
  });
  return await client.logs.create(proj.key, { name: uniqueName("log") });
};

describe("Log.useMosaicDrop", () => {
  it("should place a log layout in the target mosaic node on drop", async () => {
    const l = await createLog();
    const { result, store } = await renderHookWithConsole(() => Log.useMosaicDrop(), {
      client,
    });
    result.current({ id: log.ontologyID(l.key), nodeKey: 3, location: "top" });
    await waitFor(() => {
      const placed = Session.Layout.select(store.getState(), l.key);
      expect(placed?.name).toBe(l.name);
      expect(placed?.type).toBe("log");
      expect(placed?.tab).toMatchObject({ mosaicKey: 3, location: "top" });
    });
  });
});
