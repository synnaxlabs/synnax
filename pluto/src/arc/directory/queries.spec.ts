// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, NotFoundError, type resource } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Arc } from "@/arc";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const uniqueName = (): string => `d_${id.create().replace(/-/g, "_")}`;

const retrieveRoot = async (): Promise<resource.ID> =>
  arc.directory.resourceID((await client.arcs.directories.retrieveRoot()).key);

const retrieveDirectories = async (key: arc.directory.Key) =>
  await client.arcs.directories.retrieve([key]);

describe("Arc.Directory", () => {
  it("should return the root directory key with useRootKey", async () => {
    const { result } = await renderHookSuspended(() => Arc.Directory.useRootKey({}), {
      wrapper,
    });
    expect(result.current).toBe(root.key);
  });

  let wrapper: FC<PropsWithChildren>;
  let root: resource.ID;

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
    const [res] = await retrieveDirectories(key);
    expect(res.name).toBe(name);
    expect(res.parent).toBe(root.key);
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
    const [res] = await retrieveDirectories(d.key);
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
    await expect(retrieveDirectories(d.key)).rejects.toSatisfy(NotFoundError.matches);
  });

  describe("useResultList", () => {
    it("should find a directory by its name", async () => {
      const d = await client.arcs.directories.create({
        parent: root.key,
        name: uniqueName(),
      });
      const { result } = renderHook(
        () => Arc.Directory.useResultList({ searchTerm: d.name }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data?.map(({ key }) => key)).toContain(d.key);
      });
    });
  });

  describe("useResultAncestors", () => {
    it("should keep the directories and the directories above them live", async () => {
      const mid = await client.arcs.directories.create({
        parent: root.key,
        name: uniqueName(),
      });
      const leaf = await client.arcs.directories.create({
        parent: mid.key,
        name: uniqueName(),
      });
      const keysOf = (directories?: arc.directory.Directory[]): arc.directory.Key[] =>
        (directories ?? []).map(({ key }) => key).sort();
      const { result } = renderHook(
        () => Arc.Directory.useResultAncestors({ keys: [leaf.key] }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(keysOf(result.current.data)).toEqual(
          [leaf.key, mid.key, root.key].sort(),
        );
      });
      const name = uniqueName();
      await act(async () => {
        await client.arcs.directories.rename(mid.key, name);
      });
      await waitFor(() =>
        expect(result.current.data).toContainEqual(
          expect.objectContaining({ key: mid.key, name }),
        ),
      );
      const dest = await client.arcs.directories.create({
        parent: root.key,
        name: uniqueName(),
      });
      await act(async () => {
        await client.arcs.directories.move(
          arc.directory.resourceID(dest.key),
          arc.directory.resourceID(mid.key),
        );
      });
      await waitFor(() =>
        expect(keysOf(result.current.data)).toEqual(
          [leaf.key, mid.key, dest.key, root.key].sort(),
        ),
      );
    });
  });
});
