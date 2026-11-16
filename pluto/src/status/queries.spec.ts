// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, status } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { expectAlways } from "@synnaxlabs/x/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Flux } from "@/flux";
import { Status } from "@/status";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const path = Status.formPath;

const client = createTestClient();

describe("Status queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("list", () => {
    it("should list the statuses with a label", async () => {
      const lbl = await client.labels.create({
        name: "Filter Label",
        color: "#000000",
      });
      const [labeled] = await client.statuses.set([
        {
          name: "Labeled",
          key: uuid.create(),
          variant: "info",
          message: "labeled",
          time: TimeStamp.now(),
        },
        {
          name: "Unlabeled",
          key: uuid.create(),
          variant: "info",
          message: "unlabeled",
          time: TimeStamp.now(),
        },
      ]);
      await client.labels.label(status.resourceID(labeled.key), [lbl.key]);
      const { result } = renderHook(
        () =>
          Flux.useList((c) => c.statuses, {
            name: "statuses",
            query: { where: { labels: [lbl.key] } },
          }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual([labeled.key]);
      });
    });

    it("should add a status whose variant changes to match the filter", async () => {
      const prefix = `variant-listen-${Date.now()}`;
      const keys = [`${prefix}-error`, `${prefix}-warning`];
      await client.statuses.set([
        {
          name: "Error Status",
          key: keys[0],
          variant: "error",
          message: "fail",
          time: TimeStamp.now(),
        },
        {
          name: "Warning Status",
          key: keys[1],
          variant: "warning",
          message: "warn",
          time: TimeStamp.now(),
        },
      ]);
      const { result } = renderHook(
        () =>
          Flux.useList((c) => c.statuses, {
            name: "statuses",
            query: { where: { key: { in: keys }, variant: { in: ["error"] } } },
          }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toEqual([keys[0]]));
      await client.statuses.set({
        name: "Warning Status",
        key: keys[1],
        variant: "error",
        message: "now an error",
        time: TimeStamp.now(),
      });
      await waitFor(() => expect(result.current.data).toEqual(keys));
    });
  });

  describe("useDelete", () => {
    it("should delete a status", async () => {
      const statusToDelete = await client.statuses.set({
        name: "To Delete",
        key: "delete-hook-test",
        variant: "error",
        message: "Will be deleted",
        time: TimeStamp.now(),
      });

      const { result } = renderHook(() => Status.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(statusToDelete.key);
      });

      await expect(client.statuses.retrieve(statusToDelete.key)).rejects.toThrow();
    });

    it("should run afterOptimistic before the delete commits", async () => {
      const stat = await client.statuses.set({
        name: "Optimistic Delete",
        key: `optimistic-delete-${id.create()}`,
        variant: "error",
        message: "Will be deleted",
        time: TimeStamp.now(),
      });
      const order: string[] = [];
      const { result } = renderHook(
        () =>
          Status.useDelete({
            afterOptimistic: ({ data }) => {
              order.push(`optimistic:${data as status.Key}`);
            },
            afterSuccess: () => {
              order.push("success");
            },
          }),
        { wrapper },
      );

      await act(async () => {
        await result.current.updateAsync(stat.key);
      });

      expect(order).toEqual([`optimistic:${stat.key}`, "success"]);
    });
  });

  describe("useStatus", () => {
    it("should retrieve a single status", async () => {
      const testStatus = await client.statuses.set({
        name: "Single Retrieve",
        key: "single-retrieve-test",
        variant: "disabled",
        message: "Test single retrieve",
        time: TimeStamp.now(),
      });

      const { result } = await renderHookSuspended(
        () => Status.use({ key: testStatus.key }),
        {
          wrapper,
        },
      );

      await waitFor(() => expect(result.current).not.toBeNull());
      expect(result.current?.key).toEqual(testStatus.key);
      expect(result.current?.name).toEqual("Single Retrieve");
      expect(result.current?.variant).toEqual("disabled");
    });

    it("should update when status changes", async () => {
      const key = "reactive-test";
      await client.statuses.set({
        name: "Original",
        key,
        variant: "info",
        message: "Original",
        time: TimeStamp.now(),
      });

      const { result } = await renderHookSuspended(() => Status.use({ key }), {
        wrapper,
      });

      await waitFor(() => expect(result.current).not.toBeNull());
      expect(result.current?.name).toEqual("Original");

      await act(async () => {
        await client.statuses.set({
          name: "Updated",
          key,
          variant: "warning",
          message: "Updated",
          time: TimeStamp.now(),
        });
      });

      await waitFor(() => expect(result.current?.name).toEqual("Updated"));
      expect(result.current?.variant).toEqual("warning");
    });
  });

  describe("useForm", () => {
    it("should initialize with default values for new status", async () => {
      const { result } = renderHook(() => Status.useForm({ query: null }), { wrapper });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const v = result.current.form.value();
      expect(v.key).toEqual("");
      expect(v.variant).toEqual("success");
      expect(v.message).toEqual("");
      expect(v.name).toEqual("");
      expect(v.description).toEqual("");
      expect(v.labels).toEqual([]);
    });

    it("should create a new status on save", async () => {
      const { result } = renderHook(() => Status.useForm({ query: null }), { wrapper });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      act(() => {
        result.current.form.set(path.name, "Form Test Status");
        result.current.form.set(path.variant, "info");
        result.current.form.set(path.message, "Created via form");
        result.current.form.set(path.description, "Test description");
      });
      act(() => {
        result.current.save();
      });
      await waitFor(() => {
        const v = result.current.form.value();
        expect(v.key).not.toEqual("");
        expect(v.name).toEqual("Form Test Status");
        expect(v.variant).toEqual("info");
        expect(v.message).toEqual("Created via form");
        expect(v.description).toEqual("Test description");
      });
    });

    it("should retrieve and populate existing status", async () => {
      const key = uuid.create();
      await client.statuses.set({
        name: "Existing Status",
        key,
        variant: "warning",
        message: "Existing message",
        description: "Existing description",
        time: TimeStamp.now(),
      });
      const { result } = await renderHookSuspended(
        () => Status.useForm({ query: { key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const v = result.current.form.value();
      expect(v.key).toEqual(key);
      expect(v.name).toEqual("Existing Status");
      expect(v.variant).toEqual("warning");
      expect(v.message).toEqual("Existing message");
      expect(v.description).toEqual("Existing description");
    });

    it("should update existing status on save", async () => {
      const key = uuid.create();
      await client.statuses.set({
        name: "Original Name",
        key,
        variant: "error",
        message: "Original",
        time: TimeStamp.now(),
      });
      const { result } = await renderHookSuspended(
        () => ({
          form: Status.useForm({ query: { key } }),
          retrieve: Status.use({ key }),
        }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current.form.variant).toEqual("success"));
      act(() => {
        result.current.form.form.set(path.name, "Updated Name");
        result.current.form.form.set(path.variant, "success");
        result.current.form.form.set(path.message, "Updated message");
      });
      act(() => {
        result.current.form.save();
      });
      await waitFor(() => {
        expect(result.current.form.variant).toEqual("success");
        expect(result.current.form.form.value().name).toEqual("Updated Name");
        expect(result.current.form.form.value().variant).toEqual("success");
        expect(result.current.form.form.value().message).toEqual("Updated message");
        expect(result.current.retrieve?.name).toEqual("Updated Name");
        expect(result.current.retrieve?.variant).toEqual("success");
        expect(result.current.retrieve?.message).toEqual("Updated message");
      });
    });

    it("should not write back a status deleted while an autosave was pending", async () => {
      const key = uuid.create();
      await client.statuses.set({
        name: "Autosave Race",
        key,
        variant: "info",
        message: "Original",
        time: TimeStamp.now(),
      });
      const { result } = await renderHookSuspended(
        () =>
          Status.useForm({
            query: { key },
            autoSave: true,
            // Holds the save open long enough for the delete to overtake it.
            autoSaveDebounce: TimeSpan.milliseconds(500),
          }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      act(() => result.current.form.set(path.name, "Edited"));
      await act(async () => await client.statuses.delete(key));
      await expectAlways(async () => {
        await expect(
          async () => await client.statuses.retrieve({ key }),
        ).rejects.toThrow(NotFoundError);
      }, 800);
    });

    it("should handle status with labels", async () => {
      const label1 = await client.labels.create({ name: "Label 1", color: "#FF0000" });
      const label2 = await client.labels.create({ name: "Label 2", color: "#00FF00" });
      const { result } = renderHook(() => Status.useForm({ query: null }), { wrapper });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      act(() => {
        result.current.form.set(path.name, "Status with Labels");
        result.current.form.set(path.variant, "info");
        result.current.form.set(path.message, "Has labels");
        result.current.form.set(path.labels, [label1.key, label2.key]);
      });
      act(() => {
        result.current.save();
      });
      await waitFor(async () => {
        expect(result.current.form.value().key).not.toEqual("");
        expect(result.current.variant).toEqual("success");
      });
      const q = await renderHookSuspended(
        () => Status.use({ key: result.current.form.value().key }),
        { wrapper },
      );
      await waitFor(() =>
        expect(q.result.current?.labels).toEqual([label1.key, label2.key]),
      );
    });

    it("should replace the labels of an existing status on save", async () => {
      const kept = await client.labels.create({ name: "Kept", color: "#FF0000" });
      const dropped = await client.labels.create({ name: "Dropped", color: "#00FF00" });
      const added = await client.labels.create({ name: "Added", color: "#0000FF" });
      const key = uuid.create();
      await client.statuses.set({
        name: "Labeled",
        key,
        variant: "info",
        message: "Labeled",
        time: TimeStamp.now(),
        labels: [kept.key, dropped.key],
      });
      const { result } = await renderHookSuspended(
        () => Status.useForm({ query: { key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      act(() => {
        result.current.form.set(path.labels, [kept.key, added.key]);
      });
      await act(async () => {
        await result.current.saveAsync();
      });
      const stored = await createTestClient().statuses.retrieve({ key });
      expect(stored.labels.toSorted()).toEqual([kept.key, added.key].toSorted());
    });

    it("should keep the status unchanged when its label change fails", async () => {
      const key = uuid.create();
      await client.statuses.set({
        name: "Unchanged",
        key,
        variant: "info",
        message: "Before",
        time: TimeStamp.now(),
      });
      const { result } = await renderHookSuspended(
        () => Status.useForm({ query: { key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      act(() => {
        result.current.form.set(path.message, "After");
        result.current.form.set(path.labels, [uuid.create()]);
      });
      await act(async () => {
        await result.current.saveAsync();
      });
      expect(result.current.variant).toEqual("error");
      const stored = await createTestClient().statuses.retrieve({ key });
      expect(stored.message).toEqual("Before");
    });

    it("should update form when status changes externally", async () => {
      const key = uuid.create();
      await client.statuses.set({
        name: "Initial",
        key,
        variant: "info",
        message: "Initial",
        time: TimeStamp.now(),
      });
      const { result } = await renderHookSuspended(
        () => Status.useForm({ query: { key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.value().name).toEqual("Initial");
      await act(async () => {
        await client.statuses.set({
          name: "External Update",
          key,
          variant: "warning",
          message: "Updated externally",
          time: TimeStamp.now(),
        });
      });
      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("External Update");
        expect(result.current.form.value().variant).toEqual("warning");
        expect(result.current.form.value().message).toEqual("Updated externally");
      });
    });

    it("should generate UUID for new status if key not provided", async () => {
      const { result } = renderHook(() => Status.useForm({ query: null }), { wrapper });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      act(() => {
        result.current.form.set(path.name, "Auto UUID Status");
        result.current.form.set(path.variant, "success");
      });
      act(() => {
        result.current.save();
      });
      await waitFor(() => {
        const key = result.current.form.value().key;
        expect(key).not.toEqual("");
        expect(key).toHaveLength(36);
      });
    });

    it("should update timestamp on save", async () => {
      const { result } = renderHook(() => Status.useForm({ query: null }), { wrapper });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const initialTime = result.current.form.value().time;
      act(() => {
        result.current.form.set(path.name, "Time Test");
        result.current.form.set(path.variant, "info");
      });
      await new Promise((r) => setTimeout(r, 10));
      act(() => {
        result.current.save();
      });
      await waitFor(() => {
        const newTime = result.current.form.value().time;
        expect(newTime.afterEq(initialTime)).toBe(true);
      });
    });

    it("should handle all status variants", async () => {
      const variants = [
        "success",
        "error",
        "warning",
        "info",
        "loading",
        "disabled",
      ] as const;
      for (const variant of variants) {
        const { result } = renderHook(() => Status.useForm({ query: null }), {
          wrapper,
        });
        await waitFor(() => expect(result.current.variant).toEqual("success"));
        act(() => {
          result.current.form.set(path.name, `${variant} Status`);
          result.current.form.set(path.variant, variant);
          result.current.form.set(path.message, `Test ${variant}`);
        });
        act(() => {
          result.current.save();
        });
        await waitFor(() => {
          const v = result.current.form.value();
          expect(v.variant).toEqual(variant);
          expect(v.key).not.toEqual("");
        });
      }
    });

    it("should handle adding and removing labels", async () => {
      const label1 = await client.labels.create({
        name: "Add Label 1",
        color: "#FF0000",
      });
      const label2 = await client.labels.create({
        name: "Add Label 2",
        color: "#00FF00",
      });
      const label3 = await client.labels.create({
        name: "Add Label 3",
        color: "#0000FF",
      });
      const key = uuid.create();
      await client.statuses.set({
        name: "Label Test Status",
        key,
        variant: "info",
        message: "Testing labels",
        time: TimeStamp.now(),
      });
      await client.labels.label(status.resourceID(key), [label1.key]);
      const { result } = await renderHookSuspended(
        () => Status.useForm({ query: { key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      await waitFor(() =>
        expect(result.current.form.value().labels).toContain(label1.key),
      );
      act(() => {
        result.current.form.set(path.labels, [label2.key, label3.key]);
      });
      act(() => {
        result.current.save();
      });
      await waitFor(async () => {
        const { labels } = await createTestClient().statuses.retrieve(key);
        expect(labels.toSorted()).toEqual([label2.key, label3.key].toSorted());
      });
    });

    it("should sync label changes from external updates", async () => {
      const label1 = await client.labels.create({
        name: "Sync Label 1",
        color: "#FF0000",
      });
      const label2 = await client.labels.create({
        name: "Sync Label 2",
        color: "#00FF00",
      });
      const key = uuid.create();
      await client.statuses.set({
        name: "Label Sync Test",
        key,
        variant: "info",
        message: "Testing label sync",
        time: TimeStamp.now(),
      });
      const { result } = await renderHookSuspended(
        () => Status.useForm({ query: { key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.value().labels).toEqual([]);
      await act(async () => {
        await client.labels.label(status.resourceID(key), [label1.key]);
      });
      await waitFor(() =>
        expect(result.current.form.value().labels).toContain(label1.key),
      );
      await act(async () => {
        await client.labels.label(status.resourceID(key), [label2.key]);
      });
      await waitFor(() => {
        const labels = result.current.form.value().labels;
        expect(labels).toContain(label1.key);
        expect(labels).toContain(label2.key);
      });
      await act(async () => {
        await client.labels.remove(status.resourceID(key), [label1.key]);
      });
      await waitFor(() => {
        const labels = result.current.form.value().labels;
        expect(labels).not.toContain(label1.key);
        expect(labels).toContain(label2.key);
      });
    });

    it("should preserve key when updating existing status", async () => {
      const key = uuid.create();
      await client.statuses.set({
        name: "Original",
        key,
        variant: "info",
        message: "Original",
        time: TimeStamp.now(),
      });
      const { result } = await renderHookSuspended(
        () => ({
          form: Status.useForm({ query: { key } }),
          retrieve: Status.use({ key }),
        }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current.form.variant).toEqual("success"));
      act(() => {
        result.current.form.form.set(path.name, "Updated");
        result.current.form.form.set(path.message, "Updated message");
      });
      act(() => {
        result.current.form.save();
      });
      await waitFor(() => {
        expect(result.current.form.variant).toEqual("success");
        expect(result.current.form.form.value().key).toEqual(key);
        expect(result.current.retrieve?.key).toEqual(key);
        expect(result.current.retrieve?.name).toEqual("Updated");
      });
    });
  });
  describe("useMultiple", () => {
    it("should retrieve multiple statuses by keys", async () => {
      const status1 = await client.statuses.set({
        name: "Retrieve Multiple 1",
        key: `retrieve-multiple-${id.create()}`,
        variant: "info",
        message: "First status",
        time: TimeStamp.now(),
      });
      const status2 = await client.statuses.set({
        name: "Retrieve Multiple 2",
        key: `retrieve-multiple-${id.create()}`,
        variant: "success",
        message: "Second status",
        time: TimeStamp.now(),
      });
      const { result } = await renderHookSuspended(
        () => Status.useMultiple({ keys: [status1.key, status2.key] }),
        { wrapper },
      );
      await waitFor(() => expect(result.current).not.toBeNull());
      expect(result.current).toHaveLength(2);
      expect(result.current?.map((s) => s.key)).toContain(status1.key);
      expect(result.current?.map((s) => s.key)).toContain(status2.key);
    });
    it("should update when a status changes", async () => {
      const status1 = await client.statuses.set({
        name: "Retrieve Multiple 1",
        key: `retrieve-multiple-${id.create()}`,
        variant: "info",
        message: "First status",
        time: TimeStamp.now(),
      });
      const status2 = await client.statuses.set({
        name: "Retrieve Multiple 2",
        key: `retrieve-multiple-${id.create()}`,
        variant: "success",
        message: "Second status",
        time: TimeStamp.now(),
      });
      const { result } = await renderHookSuspended(
        () => Status.useMultiple({ keys: [status1.key, status2.key] }),
        { wrapper },
      );
      await waitFor(() => expect(result.current).not.toBeNull());
      expect(result.current).toHaveLength(2);
      expect(result.current?.map((s) => s.key)).toContain(status1.key);
      expect(result.current?.map((s) => s.key)).toContain(status2.key);
      await act(async () => {
        await client.statuses.set({
          name: "Retrieve Multiple 1 Updated",
          key: status1.key,
          variant: "success",
          message: "First status updated",
          time: TimeStamp.now(),
        });
      });
      await waitFor(() => {
        const updated = result.current?.find((s) => s.key === status1.key);
        expect(updated?.name).toEqual("Retrieve Multiple 1 Updated");
      });
    });
  });

  describe("keyed multi-retrieve", () => {
    it("should retrieve multiple statuses from the server when none are cached", async () => {
      const status1 = await client.statuses.set({
        name: "Retrieve Multiple Direct 1",
        key: `retrieve-multiple-direct-${id.create()}`,
        variant: "info",
        message: "First status",
        time: TimeStamp.now(),
      });
      const status2 = await client.statuses.set({
        name: "Retrieve Multiple Direct 2",
        key: `retrieve-multiple-direct-${id.create()}`,
        variant: "success",
        message: "Second status",
        time: TimeStamp.now(),
      });

      const statuses = await client.statuses.retrieve({
        key: [status1.key, status2.key],
      });

      expect(statuses).toHaveLength(2);
      expect(statuses.map((s) => s.key)).toContain(status1.key);
      expect(statuses.map((s) => s.key)).toContain(status2.key);
    });

    it("should use cached statuses and only fetch missing ones", async () => {
      const status1 = await client.statuses.set({
        name: "Cached Status",
        key: `cached-status-${id.create()}`,
        variant: "info",
        message: "Cached",
        time: TimeStamp.now(),
      });
      const status2 = await client.statuses.set({
        name: "Uncached Status",
        key: `uncached-status-${id.create()}`,
        variant: "success",
        message: "Uncached",
        time: TimeStamp.now(),
      });

      await client.statuses.retrieve(status1.key);

      const statuses = await client.statuses.retrieve({
        key: [status1.key, status2.key],
      });

      expect(statuses).toHaveLength(2);
      expect(statuses.map((s) => s.key)).toContain(status1.key);
      expect(statuses.map((s) => s.key)).toContain(status2.key);
    });

    it("should return all cached statuses when all are cached", async () => {
      const status1 = await client.statuses.set({
        name: "All Cached 1",
        key: `all-cached-${id.create()}`,
        variant: "info",
        message: "All cached 1",
        time: TimeStamp.now(),
      });
      const status2 = await client.statuses.set({
        name: "All Cached 2",
        key: `all-cached-${id.create()}`,
        variant: "success",
        message: "All cached 2",
        time: TimeStamp.now(),
      });

      await client.statuses.retrieve({ key: [status1.key, status2.key] });

      const statuses = await client.statuses.retrieve({
        key: [status1.key, status2.key],
      });

      expect(statuses).toHaveLength(2);
      expect(statuses.map((s) => s.key)).toContain(status1.key);
      expect(statuses.map((s) => s.key)).toContain(status2.key);
    });
  });
});
