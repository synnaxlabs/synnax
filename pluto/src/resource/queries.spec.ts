// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, label, resource } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Resource } from "@/resource";
import { renderHookSuspended } from "@/testutil";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const wrapper = createSynnaxWrapper({ client });

const randomToken = (): string => `res${Math.random().toString(36).slice(2)}`;

const createGroup = async (name: string): Promise<resource.ID> =>
  group.resourceID((await client.groups.create({ scope: "channel", name })).key);

const createLabel = async (name: string): Promise<resource.ID> =>
  label.resourceID((await client.labels.create({ name, color: "#E87E04" })).key);

describe("Resource Queries", () => {
  describe("useList", () => {
    it("should list the matches of every type for the search term", async () => {
      const token = randomToken();
      const g = await createGroup(token);
      const l = await createLabel(`${token} pump`);
      const { result } = renderHook(() => Resource.useList({ searchDebounce: 0 }), {
        wrapper,
      });
      act(() => result.current.search(token));
      await waitFor(() =>
        expect(result.current.data).toEqual([
          resource.idToString(g),
          resource.idToString(l),
        ]),
      );
      expect(result.current.getItem(resource.idToString(l))?.name).toEqual(
        `${token} pump`,
      );
    });

    it("should search only the given types", async () => {
      const token = randomToken();
      await createGroup(token);
      const l = await createLabel(token);
      const { result } = renderHook(
        () => Resource.useList({ query: { types: ["label"] }, searchDebounce: 0 }),
        { wrapper },
      );
      act(() => result.current.search(token));
      await waitFor(() =>
        expect(result.current.data).toEqual([resource.idToString(l)]),
      );
    });

    it("should list the resources of the given types for an empty term", async () => {
      for (let i = 0; i < 3; i++) await createLabel(randomToken());
      // An earlier reader of the query on a shared client would size the window.
      const { result } = renderHook(
        () => Resource.useList({ query: { types: ["label"] }, pageSize: 3 }),
        { wrapper: createSynnaxWrapper({ client: createTestClient() }) },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toHaveLength(3));
      expect(result.current.data.map((k) => resource.parseID(k).type)).toEqual([
        "label",
        "label",
        "label",
      ]);
    });

    it("should add a resource created while the list is mounted", async () => {
      // The Core holds groups from other specs, so the list takes the most it can.
      const { result } = renderHook(
        () => Resource.useList({ query: { types: ["group"] }, pageSize: 1000 }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.answered).toBe(true));
      const g = await createGroup(randomToken());
      await waitFor(() =>
        expect(result.current.data).toContain(resource.idToString(g)),
      );
    });

    it("should read a resource outside the list by its key", async () => {
      const g = await client.groups.create({ scope: "channel", name: randomToken() });
      const key = resource.idToString(group.resourceID(g.key));
      const { result } = renderHook(() => Resource.useList(), { wrapper });
      const callback = vi.fn();
      const release = result.current.subscribe(callback, key);
      await waitFor(() => expect(result.current.getItem(key)?.name).toEqual(g.name));
      await client.groups.rename(g.key, "renamed");
      await waitFor(() => expect(result.current.getItem(key)?.name).toEqual("renamed"));
      expect(callback).toHaveBeenCalled();
      release();
    });
  });

  describe("useMultiple", () => {
    it("should suspend until it returns the resources with the IDs", async () => {
      const first = await createGroup(randomToken());
      const second = await createLabel(randomToken());
      const ids = [first, second];
      const { result } = await renderHookSuspended(
        () => Resource.useMultiple({ ids }),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current.map((r) => r.id)).toEqual([first, second]),
      );
    });

    it("should leave out an ID that names no resource", async () => {
      const g = await createGroup(randomToken());
      const ids = [g, group.resourceID(uuid.create())];
      const { result } = await renderHookSuspended(
        () => Resource.useMultiple({ ids }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.map((r) => r.id)).toEqual([g]));
    });
  });
});
