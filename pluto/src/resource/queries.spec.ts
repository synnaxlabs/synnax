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
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Flux } from "@/flux";
import { Resource } from "@/resource";
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
      const { result } = renderHook(() => Resource.useList(), { wrapper });
      await waitFor(async () => {
        await act(
          async () => await result.current.retrieveAsync({ searchTerm: token }),
        );
        expect(result.current.data).toEqual([
          resource.idToString(g),
          resource.idToString(l),
        ]);
      });
      expect(result.current.getItem(resource.idToString(l))?.name).toEqual(
        `${token} pump`,
      );
    });

    it("should search only the given types", async () => {
      const token = randomToken();
      await createGroup(token);
      const l = await createLabel(token);
      const { result } = renderHook(() => Resource.useList(), { wrapper });
      await waitFor(async () => {
        await act(
          async () =>
            await result.current.retrieveAsync({ searchTerm: token, types: ["label"] }),
        );
        expect(result.current.data).toEqual([resource.idToString(l)]);
      });
    });

    it("should list the resources of the given types for an empty term", async () => {
      await createLabel(randomToken());
      const expected = await client.resources.search({
        searchTerm: "",
        types: ["label"],
        limit: 3,
      });
      const { result } = renderHook(() => Resource.useList(), { wrapper });
      await act(
        async () => await result.current.retrieveAsync({ types: ["label"], limit: 3 }),
      );
      expect(result.current.data).toEqual(expected.map((r) => r.key));
    });

    it("should add a resource created while the list is mounted", async () => {
      const { result } = renderHook(() => Resource.useList(), { wrapper });
      await act(async () => await result.current.retrieveAsync({ types: ["group"] }));
      const g = await createGroup(randomToken());
      await waitFor(() =>
        expect(result.current.data).toContain(resource.idToString(g)),
      );
    });

    it("should look up a resource outside the page by its key", async () => {
      const name = randomToken();
      const l = await createLabel(name);
      const { result } = renderHook(() => Resource.useList(), { wrapper });
      act(() => {
        result.current.getItem(resource.idToString(l));
      });
      await waitFor(() =>
        expect(result.current.getItem(resource.idToString(l))?.name).toEqual(name),
      );
    });

    it("should show the cached answer of a search before the round trip", async () => {
      const token = randomToken();
      const l = await createLabel(token);
      const query: Resource.ListQuery = { searchTerm: token, types: ["label"] };
      await expect
        .poll(async () => (await client.resources.search(query)).length)
        .toEqual(1);
      const { result } = renderHook(() => Resource.useList({ initialQuery: query }), {
        wrapper,
      });
      expect(result.current.data).toEqual([resource.idToString(l)]);
    });

    it("should update a resource read by key when it is renamed", async () => {
      const g = await client.groups.create({ scope: "channel", name: randomToken() });
      const key = resource.idToString(group.resourceID(g.key));
      const { result } = renderHook(
        () => {
          const list = Resource.useList();
          return Flux.useListItem({
            subscribe: list.subscribe,
            getItem: list.getItem,
            key,
          });
        },
        { wrapper },
      );
      await waitFor(() => expect(result.current?.name).toEqual(g.name));
      await act(async () => await client.groups.rename(g.key, "renamed"));
      await waitFor(() => expect(result.current?.name).toEqual("renamed"));
    });
  });
});
