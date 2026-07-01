// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic";
import { Session } from "@/session";
import { renderLinkHook } from "@/testutil/testutil";

const client = createTestClient();

describe("Schematic.useLink", () => {
  it("should place a schematic layout for the retrieved schematic", async () => {
    const project = await client.projects.create({ name: id.create(), layout: {} });
    const schematic = await client.schematics.create(project.key, {
      name: "Pump Schematic",
    });
    const { handler, store } = renderLinkHook(Schematic.useLink);
    await handler({ client, key: schematic.key });
    expect(Session.Layout.select(store.getState(), schematic.key)?.name).toBe(
      "Pump Schematic",
    );
  });
});
