// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { project, query, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { waitFor } from "@testing-library/react";
import { assert, describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("Schematic.TABS restore", () => {
  it("should restore a deleted schematic into the given project", async () => {
    const { restore } = Schematic.TABS[schematic.TYPE_RESOURCE_ID.type];
    assert(restore != null);
    const from = await client.projects.create({ name: uniqueName("from") });
    const to = await client.projects.create({ name: uniqueName("to") });
    const s = await client.schematics.create({
      project: from.key,
      name: uniqueName("schematic"),
    });
    await client.schematics.retrieve(s.key);
    await client.schematics.delete(s.key);
    await waitFor(() =>
      expect(query.Deleted.matches(client.schematics.getCached(s.key))).toBe(true),
    );
    await restore({ client, project: to.key, resource: schematic.resourceID(s.key) });
    const restored = await createTestClient().schematics.retrieve(s.key);
    expect(restored.project).toEqual(to.key);
  });

  it("should restore a deleted schematic out of its deleted group", async () => {
    const { restore } = Schematic.TABS[schematic.TYPE_RESOURCE_ID.type];
    assert(restore != null);
    const from = await client.projects.create({ name: uniqueName("from") });
    const to = await client.projects.create({ name: uniqueName("to") });
    const g = await client.groups.create({
      parent: project.resourceID(from.key),
      name: uniqueName("group"),
    });
    const s = await client.schematics.create({
      project: from.key,
      group: g.key,
      name: uniqueName("schematic"),
    });
    await client.schematics.retrieve(s.key);
    await client.schematics.delete(s.key);
    await waitFor(() =>
      expect(query.Deleted.matches(client.schematics.getCached(s.key))).toBe(true),
    );
    await client.groups.delete(g.key);
    await restore({ client, project: to.key, resource: schematic.resourceID(s.key) });
    const restored = await createTestClient().schematics.retrieve(s.key);
    expect(restored.project).toEqual(to.key);
    expect(restored.group).toBeUndefined();
  });
});
