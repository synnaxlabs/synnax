// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, query } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { waitFor } from "@testing-library/react";
import { assert, describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";

const client = createTestClient();

describe("Arc.TABS restore", () => {
  it("should restore a deleted Arc whose compile filled its channel lists", async () => {
    const { restore } = Arc.TABS[arc.TYPE_RESOURCE_ID.type];
    assert(restore != null);
    const [input, output] = await client.channels.create([
      { name: uniqueName("restore_in"), dataType: "float64", virtual: true },
      { name: uniqueName("restore_out"), dataType: "float64", virtual: true },
    ]);
    const a = await client.arcs.create({
      name: uniqueName("restored"),
      type: "automation",
      source: { mode: "text", text: { raw: `${input.name} * 2.0 -> ${output.name}` } },
    });
    await expect
      .poll(async () => (await client.arcs.retrieve(a.key)).reads)
      .toEqual([input.key]);
    await client.arcs.delete(a.key);
    await waitFor(() =>
      expect(query.Deleted.matches(client.arcs.watch(a.key).getCached())).toBe(true),
    );
    const proj = await client.projects.create({ name: uniqueName("restore") });
    await restore({ client, project: proj.key, resource: arc.resourceID(a.key) });
    const restored = await createTestClient().arcs.retrieve(a.key);
    expect(restored.name).toEqual(a.name);
  });
});
