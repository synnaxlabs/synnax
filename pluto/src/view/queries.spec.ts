// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { createAsyncSynnaxWrapper } from "@/testutil";
import { View } from "@/view";

const client = createTestClient();
const label = uuid.create();

describe("View queries", () => {
  let controller: AbortController;
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    controller = new AbortController();
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useList", () => {
    it("should retrieve a list of views", async () => {
      const view1 = await client.views.create({
        name: "View 1",
        type: "range",
        query: {},
      });
      const view2 = await client.views.create({
        name: "View 2",
        type: "status",
        query: { variants: ["error"] },
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
      const views = await client.views.create([
        { name: "View 1", type: "range", query: {} },
        { name: "View 2", type: "status", query: {} },
      ]);
      const keys = views.map((v) => v.key);
      const { result } = renderHook(() => View.useList(), { wrapper });
      act(() => {
        result.current.retrieve({ keys, types: ["status"] });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toEqual([views[1].key]);
    });

    it("should retrieve views that already exist before the hook is mounted", async () => {
      const views = await client.views.create([
        { name: "View 1", type: "range", query: {} },
        { name: "View 2", type: "range", query: {} },
        { name: "Solo View", type: "status", query: {} },
      ]);
      const keys = views.map((v) => v.key);
      const { result } = renderHook(() => View.useList({ initialQuery: { keys } }), {
        wrapper,
      });
      await act(async () => {
        await result.current.retrieveAsync({ keys });
      });
      expect(result.current.data.length).toBe(3);
      keys.forEach((key) => expect(result.current.data).toContain(key));
    });

    it("should update the list when a view is created", async () => {
      const { result } = renderHook(() => View.useList({}), { wrapper });
      act(() => {
        result.current.retrieve({ types: ["status"] });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const newView = await client.views.create({
        name: "New View",
        type: "status",
        query: {},
      });
      await waitFor(() => expect(result.current.data).toContain(newView.key));
    });
  });
  describe("useDelete", () => {
    it("should delete a single view", async () => {
      const view = await client.views.create({
        name: "delete-single",
        type: "range",
        query: {},
      });
      const { result } = renderHook(() => View.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(view.key);
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      await expect(client.views.retrieve(view.key)).rejects.toThrow(NotFoundError);
    });
    it("should delete multiple views", async () => {
      const [view1, view2] = await client.views.create([
        { name: "delete-multi-1", type: "range", query: {} },
        { name: "delete-multi-2", type: "status", query: {} },
      ]);
      const { result } = renderHook(() => View.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync([view1.key, view2.key]);
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      await expect(client.views.retrieve(view1.key)).rejects.toThrow(NotFoundError);
      await expect(client.views.retrieve(view2.key)).rejects.toThrow(NotFoundError);
    });
  });
  describe("useForm", () => {
    it("should create a new view", async () => {
      const { result } = renderHook(() => View.useForm({ query: null }), { wrapper });
      act(() => {
        result.current.form.set("name", "new-view");
        result.current.form.set("query.hasLabels", [label]);
        result.current.save({ signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.form.value()).toMatchObject({
          name: "new-view",
          type: "range",
          query: { hasLabels: [label] },
        });
        expect(result.current.form.value().key).toBeDefined();
        expect(result.current.form.value().key).not.toEqual("");
      });
    });
    it("should retrieve and edit an existing view", async () => {
      const view = await client.views.create({
        name: "existing-view",
        type: "status",
        query: { variants: ["error"] },
      });
      const { result } = renderHook(() => View.useForm({ query: { key: view.key } }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.value()).toEqual(view);
      act(() => {
        result.current.form.set("name", "edited-view");
        result.current.form.set("query.variants", ["warning"]);
        result.current.save({ signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.form.value()).toMatchObject({
          name: "edited-view",
          query: { variants: ["warning"] },
        });
      });
    });
    it("should update the form when the view is updated externally", async () => {
      const view = await client.views.create({
        name: "existing-view",
        type: "range",
        query: {},
      });
      const { result } = renderHook(() => View.useForm({ query: { key: view.key } }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.value()).toEqual(view);
      await client.views.create({
        key: view.key,
        name: "edited-view",
        type: "range",
        query: { hasLabels: [label] },
      });
      await waitFor(() => {
        expect(result.current.form.value()).toMatchObject({
          name: "edited-view",
          query: { hasLabels: [label] },
        });
      });
    });
  });
  describe("useRename", () => {
    it("should rename a view", async () => {
      const view = await client.views.create({
        name: "existing-view",
        type: "range",
        query: {},
      });
      const { result } = renderHook(() => View.useRename(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: view.key, name: "renamed-view" });
      });
      const retrieved = await client.views.retrieve(view.key);
      expect(retrieved.name).toEqual("renamed-view");
    });
  });
});
