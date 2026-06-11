// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { Panel } from "@/panel";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("Panel queries", () => {
  let controller: AbortController;
  let wrapper: React.FC<PropsWithChildren>;

  beforeEach(async () => {
    controller = new AbortController();
    wrapper = await createAsyncSynnaxWrapper({ client });
  });
  afterEach(() => {
    controller.abort();
  });

  describe("useRetrieve", () => {
    it("should fetch a panel by key", async () => {
      const created = await client.panels.create({ name: "retrieve-target" });
      const { result } = renderHook(() => Panel.useRetrieve({ key: created.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.key).toEqual(created.key);
      expect(result.current.data?.name).toEqual("retrieve-target");
    });
  });

  describe("useList", () => {
    it("should return panels including those created beforehand", async () => {
      const p1 = await client.panels.create({ name: "list-a" });
      const p2 = await client.panels.create({ name: "list-b" });

      const { result } = renderHook(() => Panel.useList(), { wrapper });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(p1.key);
      expect(result.current.data).toContain(p2.key);
    });

    it("should expose individual panels via getItem", async () => {
      const target = await client.panels.create({ name: "get-item-target" });
      const { result } = renderHook(() => Panel.useList(), { wrapper });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const item = result.current.getItem(target.key);
      expect(item?.key).toEqual(target.key);
      expect(item?.name).toEqual("get-item-target");
    });
  });

  describe("useRename", () => {
    it("should rename an existing panel", async () => {
      const target = await client.panels.create({ name: "before-rename" });
      const { result } = renderHook(() => Panel.useRename(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: target.key, name: "after-rename" });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const fetched = await client.panels.retrieve(target.key);
      expect(fetched.name).toEqual("after-rename");
    });
  });

  describe("useDelete", () => {
    it("should delete an existing panel", async () => {
      const target = await client.panels.create({ name: "to-delete" });
      const { result } = renderHook(() => Panel.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(target.key);
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      await expect(client.panels.retrieve(target.key)).rejects.toThrow();
    });
  });

  describe("reactive sync", () => {
    it("should propagate rename through the channel listener to useRetrieve", async () => {
      const target = await client.panels.create({ name: "reactive-before" });
      const { result } = renderHook(() => Panel.useRetrieve({ key: target.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.name).toEqual("reactive-before");

      await client.panels.rename(target.key, "reactive-after");

      await waitFor(() => expect(result.current.data?.name).toEqual("reactive-after"));
    });

    it("should propagate deletes through the channel listener", async () => {
      const target = await client.panels.create({ name: "reactive-delete" });
      const { result } = renderHook(() => Panel.useList(), { wrapper });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.data).toContain(target.key));

      await client.panels.delete(target.key);

      await waitFor(() => expect(result.current.data).not.toContain(target.key));
    });
  });
});
