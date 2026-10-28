// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label } from "@synnaxlabs/client";
import {
  createTestClient,
  spyOnSend,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { color } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import {
  afterEach,
  assert,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";

import { Flux } from "@/flux";
import { Label } from "@/label";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });
  describe("useList", () => {
    it("should update an item read by key when it changes", async () => {
      const item = await client.labels.create({ name: "original", color: "#FF0000" });
      const { result } = renderHook(
        () => {
          const list = Label.useList({ useCachedList: false });
          return Flux.useListItem({
            subscribe: list.subscribe,
            getItem: list.getItem,
            key: item.key,
          });
        },
        { wrapper },
      );
      await waitFor(() => expect(result.current?.name).toEqual("original"));
      await act(async () => {
        await client.labels.create({ ...item, name: "updated" });
      });
      await waitFor(() => expect(result.current?.name).toEqual("updated"));
    });

    it("should return a list of label keys", async () => {
      const label1 = await client.labels.create({
        name: "label1",
        color: "#FF0000",
      });
      const label2 = await client.labels.create({
        name: "label2",
        color: "#00FF00",
      });

      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(label1.key);
      expect(result.current.data).toContain(label2.key);
    });

    it("should get individual labels using getItem", async () => {
      const testLabel = await client.labels.create({
        name: "testLabel",
        color: "#E774D0",
      });

      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedLabel = result.current.getItem(testLabel.key);
      expect(retrievedLabel?.key).toEqual(testLabel.key);
      expect(retrievedLabel?.name).toEqual("testLabel");
      expect(retrievedLabel?.color).toEqual(color.construct("#E774D0"));
    });

    it("should filter labels by search term", async () => {
      await client.labels.create({
        name: "ordinary_label",
        color: "#FF0000",
      });
      await client.labels.create({
        name: "special_label",
        color: "#00FF00",
      });

      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ searchTerm: "special" });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data.length).toBeGreaterThanOrEqual(1);
        expect(
          result.current.data
            .map((key: label.Key) => result.current.getItem(key)?.name)
            .includes("special_label"),
        ).toBe(true);
      });
    });

    it("should handle pagination with limit and offset", async () => {
      const labels = await Promise.all(
        Array.from({ length: 5 }).map((_, i) =>
          client.labels.create({
            name: `paginationLabel${i}`,
            color: "#0000FF",
          }),
        ),
      );

      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({
          limit: 2,
          offset: 1,
          keys: labels.map((l) => l.key),
        });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(2);
    });

    it("should update the list when a label is created", async () => {
      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const newLabel = await client.labels.create({
        name: "newLabel",
        color: "#FFFF00",
      });

      await waitFor(() => expect(result.current.data).toContain(newLabel.key));
    });

    it("should update the list when a label is updated", async () => {
      const testLabel = await client.labels.create({
        name: "original",
        color: "#FF0000",
      });

      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.getItem(testLabel.key)?.name).toEqual("original");

      const updatedLabel = await client.labels.create({
        ...testLabel,
        name: "updated",
      });

      await waitFor(() => {
        expect(result.current.getItem(updatedLabel.key)?.name).toEqual("updated");
      });
    });

    it("should remove label from list when deleted", async () => {
      const testLabel = await client.labels.create({
        name: "toDelete",
        color: "#FF0000",
      });

      const { result } = renderHook(() => Label.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data).toContain(testLabel.key);

      await client.labels.delete(testLabel.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(testLabel.key);
      });
    });
  });

  describe("retrieveLabelsOf", () => {
    it("should retrieve labels for an ontology ID", async () => {
      const label1 = await client.labels.create({
        name: "entityLabel1",
        color: "#FF0000",
      });
      const label2 = await client.labels.create({
        name: "entityLabel2",
        color: "#00FF00",
      });
      const targetLabel = await client.labels.create({
        name: "targetEntity",
        color: "#0000FF",
      });

      await client.labels.label(label.ontologyID(targetLabel.key), [
        label1.key,
        label2.key,
      ]);

      const { result } = renderHook(
        () => Label.useResultLabelsOf({ id: label.ontologyID(targetLabel.key) }).data,
        { wrapper },
      );
      await waitFor(() => expect(result.current).toBeDefined());

      expect(result.current).toHaveLength(2);
      expect(result.current?.map((l) => l.key)).toContain(label1.key);
      expect(result.current?.map((l) => l.key)).toContain(label2.key);
    });

    it("should update when labels are added to entity", async () => {
      const targetLabel = await client.labels.create({
        name: "targetForLabeling",
        color: "#0000FF",
      });

      const { result } = renderHook(
        () =>
          Label.useResultLabelsOf({
            id: label.ontologyID(targetLabel.key),
          }).data,
        { wrapper },
      );
      await waitFor(() => expect(result.current).toBeDefined());
      const initialLength = result.current?.length ?? 0;

      const newLabel = await client.labels.create({
        name: "addedLabel",
        color: "#FFFF00",
      });
      await client.labels.label(label.ontologyID(targetLabel.key), [newLabel.key]);

      await waitFor(() => {
        expect(result.current).toHaveLength(initialLength + 1);
        expect(result.current?.map((l) => l.key)).toContain(newLabel.key);
      });
    });

    it("should update when labels are removed from entity", async () => {
      const labelToRemove = await client.labels.create({
        name: "labelToRemove",
        color: "#FF0000",
      });
      const targetLabel = await client.labels.create({
        name: "targetForRemoval",
        color: "#0000FF",
      });

      await client.labels.label(label.ontologyID(targetLabel.key), [labelToRemove.key]);

      const { result } = renderHook(
        () =>
          Label.useResultLabelsOf({
            id: label.ontologyID(targetLabel.key),
          }).data,
        { wrapper },
      );
      await waitFor(() => expect(result.current).toBeDefined());
      expect(result.current?.map((l) => l.key)).toContain(labelToRemove.key);

      await client.labels.label(label.ontologyID(targetLabel.key), [], {
        replace: true,
      });

      await waitFor(() => {
        expect(result.current?.map((l) => l.key)).not.toContain(labelToRemove.key);
      });
    });

    it("should update when a label itself is updated", async () => {
      const originalLabel = await client.labels.create({
        name: "originalName",
        color: "#FF0000",
      });
      const targetLabel = await client.labels.create({
        name: "targetEntity",
        color: "#0000FF",
      });

      await client.labels.label(label.ontologyID(targetLabel.key), [originalLabel.key]);

      const { result } = renderHook(
        () =>
          Label.useResultLabelsOf({
            id: label.ontologyID(targetLabel.key),
          }).data,
        { wrapper },
      );
      await waitFor(() => expect(result.current).toBeDefined());
      expect(result.current?.find((l) => l.key === originalLabel.key)?.name).toEqual(
        "originalName",
      );

      const updatedLabel = await client.labels.create({
        ...originalLabel,
        name: "updatedName",
      });

      await waitFor(() => {
        expect(result.current?.find((l) => l.key === updatedLabel.key)?.name).toEqual(
          "updatedName",
        );
      });
    });

    it("should update when a label itself is deleted", async () => {
      const labelToDelete = await client.labels.create({
        name: "labelToDelete",
        color: "#FF0000",
      });
      const targetLabel = await client.labels.create({
        name: "targetEntity",
        color: "#0000FF",
      });

      await client.labels.label(label.ontologyID(targetLabel.key), [labelToDelete.key]);

      const { result } = renderHook(
        () =>
          Label.useResultLabelsOf({
            id: label.ontologyID(targetLabel.key),
          }).data,
        { wrapper },
      );
      await waitFor(() => expect(result.current).toBeDefined());
      expect(result.current?.map((l) => l.key)).toContain(labelToDelete.key);

      await client.labels.delete(labelToDelete.key);

      await waitFor(() => {
        expect(result.current?.map((l) => l.key)).not.toContain(labelToDelete.key);
      });
    });
    describe("retrieveMultiple", () => {
      describe("useDirect", () => {
        it("should retrieve multiple labels", async () => {
          const labels = await client.labels.create([
            { name: "label1", color: "#FF0000" },
            { name: "label2", color: "#00FF00" },
          ]);
          const { result } = await renderHookSuspended(
            () => Label.useMultiple({ keys: labels.map((l) => l.key) }),
            { wrapper },
          );
          await waitFor(() => expect(result.current).not.toBeNull());
          expect(result.current).toEqual(labels);
        });
        it("should update when a label changes", async () => {
          const labels = await client.labels.create([
            { name: "label1", color: "#FF0000" },
            { name: "label2", color: "#00FF00" },
          ]);
          const { result } = await renderHookSuspended(
            () => Label.useMultiple({ keys: labels.map((l) => l.key) }),
            { wrapper },
          );
          await waitFor(() => expect(result.current).not.toBeNull());
          expect(result.current).toEqual(labels);
          labels[0] = await client.labels.create({
            ...labels[0],
            name: "updatedLabel",
          });
          await waitFor(() => {
            expect(result.current).toEqual(expect.arrayContaining(labels));
          });
        });
      });
      describe("useList", () => {
        it("should retrieve labels", async () => {
          const labels = await client.labels.create([
            { name: "label1", color: "#FF0000" },
            { name: "label2", color: "#00FF00" },
          ]);
          const { result } = renderHook(
            () => Label.useList({ initialQuery: { keys: labels.map((l) => l.key) } }),
            { wrapper },
          );
          result.current.retrieve({});
          await result.current.retrieveAsync({
            keys: labels.map((l) => l.key),
          });
          await waitFor(() => expect(result.current.variant).toEqual("success"));
          const label = result.current.getItem(labels[0].key);
          expect(label).toEqual(labels[0]);
        });
      });
    });
  });

  describe("useCreateForm", () => {
    it("should create a new label", async () => {
      const { result } = renderHook(() => Label.useCreateForm({ query: null }), {
        wrapper,
      });
      act(() => {
        result.current.form.set("name", "newFormLabel");
        result.current.form.set("color", "#FF00FF");
        result.current.save();
      });
      await waitFor(() => expect(result.current.form.value().key).toBeDefined());
      const { key } = result.current.form.value();
      assert(key != null);
      const stored = await client.labels.retrieve(key);
      expect(stored.name).toEqual("newFormLabel");
      expect(stored.color).toEqual(color.construct("#FF00FF"));
    });

    it("should start from a blank name and a black color", () => {
      const { result } = renderHook(() => Label.useCreateForm({ query: null }), {
        wrapper,
      });
      expect(result.current.form.value()).toEqual({
        name: "",
        color: color.construct("#000000"),
      });
    });
  });

  describe("useEditFor", () => {
    it("should add and remove labels and keep the ones it does not name", async () => {
      const [kept, removed, added, target] = await client.labels.create([
        { name: "kept", color: "#FF0000" },
        { name: "removed", color: "#00FF00" },
        { name: "added", color: "#0000FF" },
        { name: "target", color: "#000000" },
      ]);
      const id = label.ontologyID(target.key);
      await client.labels.label(id, [kept.key, removed.key]);
      const { result } = renderHook(Label.useEditFor, { wrapper });
      await act(async () => {
        await result.current.updateAsync({
          id,
          added: [added.key],
          removed: [removed.key],
        });
      });
      expect(result.current.variant).toEqual("success");
      const labels = await client.labels.retrieve({ for: id });
      expect(labels.map((l) => l.key).sort()).toEqual([kept.key, added.key].sort());
    });
  });

  describe("useDelete", () => {
    it("should delete a label", async () => {
      const labelToDelete = await client.labels.create({
        name: "deleteMe",
        color: "#FF0000",
      });

      const { result } = renderHook(Label.useDelete, { wrapper });

      act(() => {
        result.current.update(labelToDelete.key);
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));

      await expect(
        async () => await client.labels.retrieve(labelToDelete.key),
      ).rejects.toThrow();
    });

    it("should run afterOptimistic before the delete commits", async () => {
      const lbl = await client.labels.create({
        name: "optimisticDelete",
        color: "#FF0000",
      });
      const order: string[] = [];
      const { result } = renderHook(
        () =>
          Label.useDelete({
            afterOptimistic: ({ data }) => {
              order.push(`optimistic:${data as label.Key}`);
            },
            afterSuccess: () => {
              order.push("success");
            },
          }),
        { wrapper },
      );

      await act(async () => {
        await result.current.updateAsync(lbl.key);
      });

      expect(order).toEqual([`optimistic:${lbl.key}`, "success"]);
    });

    it("should handle delete operations in sequence", async () => {
      const label1 = await client.labels.create({
        name: "delete1",
        color: "#FF0000",
      });
      const label2 = await client.labels.create({
        name: "delete2",
        color: "#00FF00",
      });

      const { result: result1 } = renderHook(Label.useDelete, { wrapper });
      const { result: result2 } = renderHook(Label.useDelete, { wrapper });

      act(() => {
        result1.current.update(label1.key);
      });
      await waitFor(() => expect(result1.current.variant).toEqual("success"));

      act(() => {
        result2.current.update(label2.key);
      });
      await waitFor(() => expect(result2.current.variant).toEqual("success"));

      await expect(
        async () => await client.labels.retrieve(label1.key),
      ).rejects.toThrow();
      await expect(
        async () => await client.labels.retrieve(label2.key),
      ).rejects.toThrow();
    });
  });
});

describe("useForm", () => {
  const formClient = createTestClient();
  let wrapper: FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client: formClient });
    await waitForStreamLive(formClient.connection);
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  const createLabel = async (): Promise<label.Label> =>
    await client.labels.create({ name: "Nominal", color: "#00FF00" });

  const renderForm = async (key: label.Key) =>
    await renderHookSuspended(() => Label.useForm({ query: { key } }), { wrapper });

  it("should show the cached label", async () => {
    const lbl = await createLabel();
    const { result } = await renderForm(lbl.key);
    expect(result.current.form.value()).toEqual({
      name: "Nominal",
      color: color.construct("#00FF00"),
    });
  });

  it("should send an edit to the name as a path action", async () => {
    const lbl = await createLabel();
    const { result } = await renderForm(lbl.key);
    const send = spyOnSend(formClient);
    act(() => result.current.form.set("name", "Anomaly"));
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(send.mock.calls.map(([target]) => target)).toContain("/label/dispatch");
    expect((await createTestClient().labels.retrieve(lbl.key)).name).toBe("Anomaly");
  });

  it("should send an edit to the color", async () => {
    const lbl = await createLabel();
    const { result } = await renderForm(lbl.key);
    act(() => result.current.form.set("color", color.construct("#FF0000")));
    await act(async () => expect(await result.current.flush()).toBe(true));
    const stored = await createTestClient().labels.retrieve(lbl.key);
    expect(stored.color).toEqual(color.construct("#FF0000"));
  });

  it("should hold an empty name without sending it", async () => {
    const lbl = await createLabel();
    const { result } = await renderForm(lbl.key);
    act(() => result.current.form.set("name", ""));
    await act(async () => expect(await result.current.flush()).toBe(false));
    expect((await createTestClient().labels.retrieve(lbl.key)).name).toBe("Nominal");
  });

  it("should show a rename from another client and keep the color", async () => {
    const lbl = await createLabel();
    const { result } = await renderForm(lbl.key);
    const prevColor = result.current.form.get("color").value;
    await act(async () => {
      await client.labels.dispatch(
        lbl.key,
        label.set({ path: ["name"], value: "Anomaly" }),
      );
    });
    await waitFor(() => expect(result.current.form.value().name).toBe("Anomaly"));
    expect(result.current.form.get("color").value).toBe(prevColor);
  });
});
