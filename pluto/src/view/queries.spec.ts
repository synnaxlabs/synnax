// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, view } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
} from "@synnaxlabs/client/testutil";
import { deep, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Flux } from "@/flux";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { View } from "@/view";

const path = View.formPath;

const client = createTestClient();
const label = uuid.create();

describe("View queries", () => {
  let controller: AbortController;
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    controller = new AbortController();
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("list", () => {
    it("should list the views of a type", async () => {
      const prefix = uuid.create();
      const [range, status] = await client.views.create([
        { name: `${prefix}-range`, type: "range" },
        { name: `${prefix}-status`, type: "status" },
      ]);
      const { result } = renderHook(
        () =>
          Flux.useList((c) => c.views, {
            name: "views",
            query: {
              where: { name: { in: [range.name, status.name] }, type: { eq: "range" } },
            },
          }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual([range.key]);
      });
    });
  });
  describe("useResult", () => {
    it("should follow a field that another client sets", async () => {
      const created = await client.views.create({
        name: "sorted-view",
        type: "range",
        sort: "newest",
      });
      const { result } = renderHook(() => View.useResult({ key: created.key }), {
        wrapper,
      });
      await waitFor(() =>
        expect(result.current.data).toMatchObject({ sort: "newest" }),
      );
      const remote = createTestClient();
      await remote.views.retrieve(created.key);
      await remote.views.dispatch(
        created.key,
        view.set({ path: ["sort"], value: "name" }),
      );
      await waitFor(() => expect(result.current.data).toMatchObject({ sort: "name" }));
    });
  });

  describe("useDelete", () => {
    it("should delete a single view", async () => {
      const view = await client.views.create({
        name: "delete-single",
        type: "range",
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
        { name: "delete-multi-1", type: "range" },
        { name: "delete-multi-2", type: "status" },
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
        result.current.form.set(path.name, "new-view");
        result.current.form.set(path.where, { labels: [label] });
        result.current.save({ signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.form.value()).toMatchObject({
          name: "new-view",
          type: "range",
          where: { labels: [label] },
        });
        expect(result.current.form.value().key).toBeDefined();
        expect(result.current.form.value().key).not.toEqual("");
      });
    });
    it("should retrieve and edit an existing view", async () => {
      const view = await client.views.create({
        name: "existing-view",
        type: "status",
        where: { variant: { in: ["error"] } },
      });
      const { result } = renderHook(() => View.useForm({ query: { key: view.key } }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.value()).toEqual(view);
      act(() => {
        result.current.form.set(path.name, "edited-view");
        result.current.form.set(
          deep.present(deep.variant(path, "type", "status").where).variant,
          {
            in: ["warning"],
          },
        );
        result.current.save({ signal: controller.signal });
      });
      await waitFor(async () => {
        expect(result.current.form.value()).toMatchObject({
          name: "edited-view",
          where: { variant: { in: ["warning"] } },
        });
        expect(await client.views.retrieve(view.key)).toEqual({
          ...view,
          name: "edited-view",
          where: { variant: { in: ["warning"] } },
        });
      });
    });
    it("should update the form when the view is updated externally", async () => {
      const existing = await client.views.create({
        name: "existing-view",
        type: "range",
      });
      const { result } = renderHook(
        () => View.useForm({ query: { key: existing.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.value()).toEqual(existing);
      await client.views.dispatch(existing.key, [
        view.set({ path: ["name"], value: "edited-view" }),
        view.set({ path: ["where"], value: { labels: [label] } }),
      ]);
      await waitFor(() => {
        expect(result.current.form.value()).toMatchObject({
          name: "edited-view",
          where: { labels: [label] },
        });
      });
    });
    it("should save a name back after another client changes it", async () => {
      const view = await client.views.create({ name: "existing-view", type: "range" });
      const { result } = renderHook(() => View.useForm({ query: { key: view.key } }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      await createTestClient().views.rename(view.key, "remote-view");
      await waitFor(() =>
        expect(result.current.form.value().name).toEqual("remote-view"),
      );
      act(() => result.current.form.set(path.name, "existing-view"));
      await act(async () => expect(await result.current.saveAsync()).toBe(true));
      const stored = await createTestClient().views.retrieve(view.key);
      expect(stored.name).toEqual("existing-view");
    });
    it("should keep typed values and show the error on a refused save", async () => {
      const view = await client.views.create({ name: "refused-view", type: "range" });
      const viewer = await createTestClientWithRole(client, "Viewer");
      const { result } = await renderHookSuspended(
        () => View.useForm({ query: { key: view.key } }),
        { wrapper: await createAsyncSynnaxWrapper({ client: viewer }) },
      );
      act(() => result.current.form.set(path.name, "typed-view"));
      await act(async () => expect(await result.current.saveAsync()).toBe(false));
      expect(result.current.variant).toEqual("error");
      expect(result.current.form.value().name).toEqual("typed-view");
    });
    it("should refuse a name made only of spaces", async () => {
      const { result } = renderHook(() => View.useForm({ query: null }), { wrapper });
      act(() => result.current.form.set(path.name, "   "));
      await act(async () => expect(await result.current.saveAsync()).toBe(false));
      const { status } = result.current.form.get(path.name);
      expect(status.variant).toEqual("error");
      expect(status.message).toEqual("Name is required");
    });
    it("should trim the name of a new view before it saves", async () => {
      const { result } = renderHook(() => View.useForm({ query: null }), { wrapper });
      act(() => result.current.form.set(path.name, "  trimmed-view  "));
      await act(async () => expect(await result.current.saveAsync()).toBe(true));
      const { key } = result.current.form.value();
      if (key == null) throw new Error("the view has no key");
      const stored = await createTestClient().views.retrieve(key);
      expect(stored.name).toEqual("trimmed-view");
    });
    it("should trim the name of an existing view before it saves", async () => {
      const view = await client.views.create({ name: "existing-view", type: "range" });
      const { result } = renderHook(() => View.useForm({ query: { key: view.key } }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      act(() => result.current.form.set(path.name, "  renamed-view  "));
      await act(async () => expect(await result.current.saveAsync()).toBe(true));
      const stored = await createTestClient().views.retrieve(view.key);
      expect(stored.name).toEqual("renamed-view");
    });
    it("should save a view it starts from without a query", async () => {
      const view = await client.views.create({ name: "existing-view", type: "range" });
      const { result } = renderHook(
        () => View.useForm({ query: null, initialValues: view }),
        { wrapper },
      );
      act(() => result.current.form.set(path.where, { labels: [label] }));
      await act(async () => expect(await result.current.saveAsync()).toBe(true));
      const stored = await createTestClient().views.retrieve(view.key);
      expect(stored).toEqual({ ...view, where: { labels: [label] } });
    });
  });
  describe("useRename", () => {
    it("should rename a view", async () => {
      const view = await client.views.create({
        name: "existing-view",
        type: "range",
      });
      const { result } = renderHook(() => View.useRename(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: view.key, name: "renamed-view" });
      });
      const retrieved = await client.views.retrieve(view.key);
      expect(retrieved.name).toEqual("renamed-view");
    });
  });
  describe("useSetFields", () => {
    it("should set the icon and color of a view", async () => {
      const created = await client.views.create({ name: "styled-view", type: "range" });
      const { result } = renderHook(() => View.useSetFields(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({
          key: created.key,
          actions: [
            view.set({ path: ["icon"], value: "Range" }),
            view.set({ path: ["color"], value: "#FF0000" }),
          ],
        });
      });
      const retrieved = await createTestClient().views.retrieve(created.key);
      expect(retrieved.icon).toEqual("Range");
      expect(retrieved.color).toEqual([255, 0, 0, 1]);
    });
  });
});
