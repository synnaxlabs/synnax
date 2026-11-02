// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, NotFoundError, ontology } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Arc } from "@/arc";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const uniqueName = (): string => `d_${id.create().replace(/-/g, "_")}`;

const retrieveRoot = async (): Promise<ontology.ID> => {
  const [root] = await client.ontology.children.retrieve({
    ids: ontology.ROOT_ID,
    types: ["arc_directory"],
  });
  return root.id;
};

describe("Arc.Directory", () => {
  let wrapper: FC<PropsWithChildren>;
  let root: ontology.ID;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    root = await retrieveRoot();
  });

  it("should create a directory with useCreate", async () => {
    const { result } = renderHook(() => Arc.Directory.useCreate(), { wrapper });
    const key = uuid.create();
    const name = uniqueName();
    await act(async () => {
      await result.current.updateAsync({ parent: root.key, key, name });
    });
    const res = await client.ontology.retrieve(arc.directory.ontologyID(key));
    expect(res.name).toBe(name);
  });

  it("should rename a directory with useRename", async () => {
    const d = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const { result } = renderHook(() => Arc.Directory.useRename(), { wrapper });
    const name = uniqueName();
    await act(async () => {
      await result.current.updateAsync({ key: d.key, name });
    });
    const res = await client.ontology.retrieve(arc.directory.ontologyID(d.key));
    expect(res.name).toBe(name);
  });

  it("should delete a directory with useDelete", async () => {
    const d = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const { result } = renderHook(() => Arc.Directory.useDelete(), { wrapper });
    await act(async () => {
      await result.current.updateAsync(d.key);
    });
    await expect(
      client.ontology.retrieve(arc.directory.ontologyID(d.key)),
    ).rejects.toSatisfy(NotFoundError.matches);
  });

  it("should move an Arc into a directory with useMove", async () => {
    const d = await client.arcs.directories.create({
      parent: root.key,
      name: uniqueName(),
    });
    const a = await client.arcs.create({ name: uniqueName(), mode: "text" });
    const destination = arc.directory.ontologyID(d.key);
    const { result } = renderHook(() => Arc.Directory.useMove(), { wrapper });
    await act(async () => {
      await result.current.updateAsync({
        source: root,
        destination,
        ids: [arc.ontologyID(a.key)],
      });
    });
    const parents = await client.ontology.parents.retrieve({
      ids: arc.ontologyID(a.key),
    });
    expect(parents.map((p) => p.id)).toEqual([destination]);
  });
});
