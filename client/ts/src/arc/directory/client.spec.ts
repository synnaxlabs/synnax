// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { arc } from "@/arc";
import { NotFoundError, ValidationError } from "@/index";
import { ontology } from "@/ontology";
import { type resource } from "@/resource";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const uniqueName = (): string => `d_${id.create().replace(/-/g, "_")}`;

const retrieveRoot = async (): Promise<resource.ID> => {
  const children = await client.ontology.children.retrieve({
    ids: ontology.ROOT_ID,
    types: ["arc_directory"],
  });
  expect(children).toHaveLength(1);
  return children[0].id;
};

describe("arc.directory", () => {
  it("should create a directory in its parent", async () => {
    const root = await retrieveRoot();
    const d = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const children = await client.ontology.children.retrieve({ ids: root });
    expect(children.map((c) => c.id.key)).toContain(d.key);
  });

  it("should rename a directory", async () => {
    const root = await retrieveRoot();
    const d = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const name = uniqueName();
    await client.arcs.directories.rename(d.key, name);
    const res = await client.ontology.retrieve(arc.directory.resourceID(d.key));
    expect(res.name).toBe(name);
  });

  it("should delete a directory", async () => {
    const root = await retrieveRoot();
    const d = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    await client.arcs.directories.delete(d.key);
    await expect(
      client.ontology.retrieve(arc.directory.resourceID(d.key)),
    ).rejects.toSatisfy(NotFoundError.matches);
  });

  it("should move an Arc into a directory", async () => {
    const root = await retrieveRoot();
    const d = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const a = await client.arcs.create({
      name: uniqueName(),
      type: "automation",
      source: { mode: "text" },
    });
    const dirID = arc.directory.resourceID(d.key);
    await client.arcs.directories.move(root, dirID, arc.resourceID(a.key));
    const parents = await client.ontology.parents.retrieve({
      ids: arc.resourceID(a.key),
    });
    expect(parents.map((p) => p.id)).toEqual([dirID]);
  });

  it("should reject a move whose source does not hold the Arc", async () => {
    const root = await retrieveRoot();
    const named = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const dest = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const a = await client.arcs.create({
      name: uniqueName(),
      type: "automation",
      source: { mode: "text" },
    });
    await expect(
      client.arcs.directories.move(
        arc.directory.resourceID(named.key),
        arc.directory.resourceID(dest.key),
        arc.resourceID(a.key),
      ),
    ).rejects.toSatisfy(ValidationError.matches);
    const parents = await client.ontology.parents.retrieve({
      ids: arc.resourceID(a.key),
    });
    expect(parents.map((p) => p.id)).toEqual([root]);
  });
});
