// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, NotFoundError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Flux } from "@/flux";
import { Group } from "@/group";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("Group queries", () => {
  let controller: AbortController;
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  beforeEach(() => {
    controller = new AbortController();
  });

  afterEach(() => {
    controller.abort();
  });

  describe("list", () => {
    const useGroups = (query: group.RetrieveMultipleParams) =>
      Flux.useList((c) => c.groups, {
        name: "groups",
        query: { ...query, order: { field: "name" } },
      });

    it("should list the groups of a parent in name order", async () => {
      const parent = await client.groups.create({ scope: "channel", name: "parent" });
      const b = await client.groups.create({
        parent: group.resourceID(parent.key),
        name: "b",
      });
      const a = await client.groups.create({
        parent: group.resourceID(parent.key),
        name: "a",
      });
      const { result } = renderHook(
        () => useGroups({ parents: [group.resourceID(parent.key)] }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual([a.key, b.key]);
      });
      expect(result.current.getItem(a.key)?.name).toBe("a");
    });

    it("should keep only the top-level groups of the queried scope", async () => {
      const top = await client.groups.create({
        scope: "channel",
        name: `top-${uuid.create()}`,
      });
      const nested = await client.groups.create({
        parent: group.resourceID(top.key),
        name: `nested-${uuid.create()}`,
      });
      const other = await client.groups.create({
        scope: "project",
        name: `other-${uuid.create()}`,
      });
      const { result } = renderHook(
        () =>
          useGroups({
            scope: "channel",
            nested: false,
            name: [top.name, nested.name, other.name],
          }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toEqual([top.key]));
    });

    it("should move a group between the lists of its parents", async () => {
      const from = await client.groups.create({ scope: "channel", name: "from" });
      const to = await client.groups.create({ scope: "channel", name: "to" });
      const g = await client.groups.create({
        parent: group.resourceID(from.key),
        name: "moving",
      });
      const { result } = renderHook(
        () => ({
          from: useGroups({ parents: [group.resourceID(from.key)] }),
          to: useGroups({ parents: [group.resourceID(to.key)] }),
        }),
        { wrapper },
      );
      act(() => {
        result.current.from.more();
        result.current.to.more();
      });
      await waitFor(() => {
        expect(result.current.from.data).toEqual([g.key]);
        expect(result.current.to.answered).toBe(true);
      });
      expect(result.current.to.data).toEqual([]);
      await act(async () => {
        await client.groups.setParent(g.key, group.resourceID(to.key));
      });
      await waitFor(() => {
        expect(result.current.from.data).toEqual([]);
        expect(result.current.to.data).toEqual([g.key]);
      });
    });
  });

  describe("useCreate", () => {
    it("should create a new group", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: "test-parent-create",
      });

      const { result } = renderHook(() => Group.useCreate(), {
        wrapper,
      });

      await act(async () => {
        await result.current.updateAsync({
          key: uuid.create(),
          name: "created-group",
          parent: group.resourceID(parent.key),
        });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toBeDefined();
        expect(result.current.data?.name).toBe("created-group");
      });

      const retrieved = await client.groups.retrieve(
        result.current.data?.key as string,
      );
      expect(retrieved.name).toBe("created-group");
    });

    it("should refuse a group whose key exists", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: "test-parent-key",
      });
      const g = await client.groups.create({
        parent: group.resourceID(parent.key),
        name: "original-name",
      });

      const { result } = renderHook(() => Group.useCreate(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          key: g.key,
          name: "updated-name",
          parent: group.resourceID(parent.key),
        });
      });

      expect(result.current.variant).toBe("error");
      expect(result.current.status.description).toContain(
        `group ${g.key} already exists`,
      );
      const retrieved = await client.groups.retrieve(g.key);
      expect(retrieved.name).toBe("original-name");
    });
  });

  describe("useRename", () => {
    it("should rename an existing group", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: "test-parent-rename",
      });
      const testGroup = await client.groups.create({
        parent: group.resourceID(parent.key),
        name: "original-name",
      });

      const { result } = renderHook(() => Group.useRename(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: testGroup.key, name: "new-name" });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const retrieved = await client.groups.retrieve(testGroup.key);
      expect(retrieved.name).toBe("new-name");
    });
  });

  describe("useDelete", () => {
    it("should delete an existing group", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: "test-parent-delete",
      });
      const testGroup = await client.groups.create({
        parent: group.resourceID(parent.key),
        name: "to-be-deleted",
      });

      const { result } = renderHook(() => Group.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: testGroup.key });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(client.groups.retrieve(testGroup.key)).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should run afterOptimistic before the delete commits", async () => {
      const testGroup = await client.groups.create({
        scope: "channel",
        name: "optimistic-delete",
      });
      const order: string[] = [];
      const { result } = renderHook(
        () =>
          Group.useDelete({
            afterOptimistic: ({ data }) => {
              order.push(`optimistic:${data.key}`);
            },
            afterSuccess: () => {
              order.push("success");
            },
          }),
        { wrapper },
      );

      await act(async () => {
        await result.current.updateAsync({ key: testGroup.key });
      });

      expect(order).toEqual([`optimistic:${testGroup.key}`, "success"]);
    });
  });

  describe("useResultAncestors", () => {
    it("should keep the groups and the groups above them live", async () => {
      const root = await client.groups.create({ scope: "channel", name: "root" });
      const mid = await client.groups.create({
        parent: group.resourceID(root.key),
        name: "mid",
      });
      const leaf = await client.groups.create({
        parent: group.resourceID(mid.key),
        name: "leaf",
      });
      const keysOf = (groups?: group.Group[]): group.Key[] =>
        (groups ?? []).map(({ key }) => key).sort();
      const { result } = renderHook(
        () => Group.useResultAncestors({ keys: [leaf.key] }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(keysOf(result.current.data)).toEqual(
          [leaf.key, mid.key, root.key].sort(),
        );
      });
      await act(async () => {
        await client.groups.rename(root.key, "renamed");
      });
      await waitFor(() =>
        expect(result.current.data).toContainEqual(
          expect.objectContaining({ key: root.key, name: "renamed" }),
        ),
      );
      await act(async () => {
        await client.groups.setParent(mid.key, null);
      });
      await waitFor(() =>
        expect(keysOf(result.current.data)).toEqual([leaf.key, mid.key].sort()),
      );
    });
  });
});
