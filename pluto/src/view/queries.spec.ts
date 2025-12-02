// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { View } from "@/view";

const client = createTestClient();

describe("View queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useList", () => {
    it("should retrieve a list of views", async () => {
      const view1 = await client.views.create({
        name: "View 1",
        type: "lineplot",
        query: { channels: ["ch1"] },
      });
      const view2 = await client.views.create({
        name: "View 2",
        type: "table",
        query: { channels: ["ch2"] },
      });

      const { result } = renderHook(() => View.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(view1.key);
      expect(result.current.data).toContain(view2.key);
    });

    it("should retrieve a list of views by type", async () => {
      const type = id.create();
      const view1 = await client.views.create({
        name: "View 1",
        type,
        query: { channels: ["ch1"] },
      });
      const view2 = await client.views.create({
        name: "View 2",
        type,
        query: { channels: ["ch2"] },
      });

      const { result } = renderHook(() => View.useList(), { wrapper });
      act(() => {
        result.current.retrieve({ types: [type] });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBe(2);
      expect(result.current.data).toContain(view1.key);
      expect(result.current.data).toContain(view2.key);
    });

    it("should retrieve views that already exist before the hook is mounted", async () => {
      const type = id.create();
      const views = await client.views.create([
        {
          name: "View 1",
          type,
          query: { channels: ["ch1"] },
        },
        {
          name: "View 2",
          type,
          query: { channels: ["ch2"] },
        },
      ]);
      const soloView = await client.views.create({
        name: "Solo View",
        type,
        query: { channels: ["ch2"] },
      });
      const { result } = renderHook(
        () => View.useList({ initialQuery: { types: [type] } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.data.length).toBe(3));
      expect(result.current.data).toContain(views[0].key);
      expect(result.current.data).toContain(views[1].key);
      expect(result.current.data).toContain(soloView.key);
    });

    it("should update the list when a view is created", async () => {
      const type = id.create();
      const { result } = renderHook(() => View.useList({}), { wrapper });
      act(() => {
        result.current.retrieve({ types: [type] });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBe(0);
      const newView = await client.views.create({
        name: "New View",
        type,
        query: { channels: ["ch3"] },
      });
      await waitFor(() => {
        expect(result.current.data.length).toBe(1);
        expect(result.current.data).toContain(newView.key);
      });
    });
  });
  describe("useDelete", () => {
    it("should delete a single view", async () => {
      const view = await client.views.create({
        name: "delete-single",
        type: "lineplot",
        query: { channels: ["ch1"] },
      });
      const { result } = renderHook(() => View.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(view.key);
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      await expect(client.views.retrieve({ key: view.key })).rejects.toThrow();
    });
    it("should delete multiple views", async () => {
      const view1 = await client.views.create({
        name: "delete-multi-1",
        type: "lineplot",
        query: { channels: ["ch1"] },
      });
      const view2 = await client.views.create({
        name: "delete-multi-2",
        type: "lineplot",
        query: { channels: ["ch2"] },
      });
      const { result } = renderHook(() => View.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync([view1.key, view2.key]);
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      await expect(client.views.retrieve({ key: view1.key })).rejects.toThrow();
      await expect(client.views.retrieve({ key: view2.key })).rejects.toThrow();
    });
  });
});
