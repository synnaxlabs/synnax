// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, NotFoundError } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { View } from "@/view";

const client = createTestClient();

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
      await act(async () => {
        await result.current.retrieveAsync({ types: [type] });
      });
      expect(result.current.data.length).toBe(3);
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
      await expect(client.views.retrieve({ key: view.key })).rejects.toThrow(
        NotFoundError,
      );
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
      await expect(client.views.retrieve({ key: view1.key })).rejects.toThrow(
        NotFoundError,
      );
      await expect(client.views.retrieve({ key: view2.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });
  describe("useForm", () => {
    it("should create a new view", async () => {
      const { result } = renderHook(() => View.useForm({ query: {} }), { wrapper });
      act(() => {
        result.current.form.set("name", "new-view");
        result.current.form.set("type", "lineplot");
        result.current.form.set("query", { channels: ["ch1"] });
        result.current.save({ signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("new-view");
        expect(result.current.form.value().type).toEqual("lineplot");
        expect(result.current.form.value().query).toEqual({ channels: ["ch1"] });
        expect(result.current.form.value().key).toBeDefined();
        expect(result.current.form.value().key).not.toEqual("");
      });
    });
    it("should retrieve and edit an existing view", async () => {
      const view = await client.views.create({
        name: "existing-view",
        type: "lineplot",
        query: { channels: ["ch1"] },
      });
      const { result } = renderHook(() => View.useForm({ query: { key: view.key } }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.value().name).toEqual("existing-view");
      expect(result.current.form.value().type).toEqual("lineplot");
      expect(result.current.form.value().query).toEqual({ channels: ["ch1"] });
      expect(result.current.form.value().key).toEqual(view.key);
      act(() => {
        result.current.form.set("name", "edited-view");
        result.current.form.set("query", { channels: ["ch2"] });
        result.current.save({ signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("edited-view");
        expect(result.current.form.value().query).toEqual({ channels: ["ch2"] });
      });
    });
    it("should update the form when the view is updated externally", async () => {
      const view = await client.views.create({
        name: "existing-view",
        type: "lineplot",
        query: { channels: ["ch1"] },
      });
      const { result } = renderHook(() => View.useForm({ query: { key: view.key } }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.value().name).toEqual("existing-view");
      expect(result.current.form.value().type).toEqual("lineplot");
      expect(result.current.form.value().query).toEqual({ channels: ["ch1"] });
      expect(result.current.form.value().key).toEqual(view.key);
      await client.views.create({
        key: view.key,
        name: "edited-view",
        type: "lineplot",
        query: { channels: ["ch2"] },
      });
      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("edited-view");
        expect(result.current.form.value().query).toEqual({ channels: ["ch2"] });
      });
    });
  });
  describe("useRename", () => {
    it("should rename a view", async () => {
      const view = await client.views.create({
        name: "existing-view",
        type: "lineplot",
        query: { channels: ["ch1"] },
      });
      const { result } = renderHook(() => View.useRename(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: view.key, name: "renamed-view" });
      });
      const retrieved = await client.views.retrieve({ key: view.key });
      expect(retrieved.name).toEqual("renamed-view");
    });
  });
});
