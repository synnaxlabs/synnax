// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, query, status } from "@synnaxlabs/client";
import {
  createTestClient,
  spyOnSend,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { color, id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
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

const path = Label.createFormPath;

const client = createTestClient();

describe("queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });
  describe("list", () => {
    it("should list the matching labels in name order", async () => {
      const [b, a] = await client.labels.create([
        { name: "b", color: "#FF0000" },
        { name: "a", color: "#00FF00" },
      ]);
      const { result } = renderHook(
        () =>
          Flux.useList((c) => c.labels, {
            name: "labels",
            query: { where: { key: { in: [a.key, b.key] } }, order: { field: "name" } },
          }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual([a.key, b.key]);
      });
      expect(result.current.getItem(a.key)).toEqual(a);
    });

    it("should list the labels that match a search term", async () => {
      const name = `special-${id.create()}`;
      const match = await client.labels.create({ name, color: "#00FF00" });
      const { result } = renderHook(
        () => Flux.useList((c) => c.labels, { name: "labels", searchDebounce: 0 }),
        { wrapper },
      );
      act(() => result.current.search(name));
      await waitFor(() => expect(result.current.data).toContain(match.key));
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
      it("should answer an empty key list without a request", async () => {
        const send = spyOnSend(client);
        const { result } = await renderHookSuspended(
          () => Label.useMultiple({ keys: [] }),
          { wrapper },
        );
        expect(result.current).toEqual([]);
        expect(send.mock.calls.filter(([t]) => t === "/label/retrieve")).toEqual([]);
        send.mockRestore();
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
        await client.labels.dispatch(
          labels[0].key,
          label.set({ path: ["name"], value: "updatedLabel" }),
        );
        labels[0] = { ...labels[0], name: "updatedLabel" };
        await waitFor(() => {
          expect(result.current).toEqual(expect.arrayContaining(labels));
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
        result.current.form.set(path.name, "newFormLabel");
        result.current.form.set(path.color, color.construct("#FF00FF"));
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
    const createLabels = async () =>
      await client.labels.create([
        { name: "kept", color: "#FF0000" },
        { name: "removed", color: "#00FF00" },
        { name: "added", color: "#0000FF" },
      ]);

    it("should edit the labels of a status and keep the ones it does not name", async () => {
      const [kept, removed, added] = await createLabels();
      const st = await client.statuses.set({
        variant: "info",
        message: "labeled",
        labels: [kept.key, removed.key],
      });
      const { result } = renderHook(Label.useEditFor, { wrapper });
      await act(async () => {
        await result.current.updateAsync({
          id: status.resourceID(st.key),
          added: [added.key],
          removed: [removed.key],
        });
      });
      expect(result.current.variant).toEqual("success");
      const stored = await createTestClient().statuses.retrieve(st.key);
      expect(stored.labels.toSorted()).toEqual([kept.key, added.key].toSorted());
    });

    it("should edit the labels of a range and reach a client that caches it", async () => {
      const [kept, removed, added] = await createLabels();
      const rng = await client.ranges.create({
        name: "labeled",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        labels: [kept.key, removed.key],
      });
      const observer = createTestClient();
      await observer.connect();
      await waitForStreamLive(observer.connection);
      await observer.ranges.retrieve(rng.key);
      const { result } = renderHook(Label.useEditFor, { wrapper });
      await act(async () => {
        await result.current.updateAsync({
          id: rng.resourceID,
          added: [added.key],
          removed: [removed.key],
        });
      });
      expect(result.current.variant).toEqual("success");
      const expected = [kept.key, added.key].toSorted();
      await waitFor(() => {
        const cached = observer.ranges.watch(rng.key).getCached();
        expect(query.isLive(cached) && cached.labels.toSorted()).toEqual(expected);
      });
    });

    it("should fail for a resource that cannot hold labels", async () => {
      const [kept, , added] = await createLabels();
      const { result } = renderHook(Label.useEditFor, { wrapper });
      await act(async () => {
        expect(
          await result.current.updateAsync({
            id: label.resourceID(kept.key),
            added: [added.key],
            removed: [],
          }),
        ).toBe(false);
      });
      expect(result.current.variant).toEqual("error");
      expect(result.current.status.description).toContain("a label cannot have labels");
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
    act(() => result.current.form.set(path.name, "Anomaly"));
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(send.mock.calls.map(([target]) => target)).toContain("/label/dispatch");
    expect((await createTestClient().labels.retrieve(lbl.key)).name).toBe("Anomaly");
  });

  it("should send an edit to the color", async () => {
    const lbl = await createLabel();
    const { result } = await renderForm(lbl.key);
    act(() => result.current.form.set(path.color, color.construct("#FF0000")));
    await act(async () => expect(await result.current.flush()).toBe(true));
    const stored = await createTestClient().labels.retrieve(lbl.key);
    expect(stored.color).toEqual(color.construct("#FF0000"));
  });

  it("should hold an empty name without sending it", async () => {
    const lbl = await createLabel();
    const { result } = await renderForm(lbl.key);
    act(() => result.current.form.set(path.name, ""));
    await act(async () => expect(await result.current.flush()).toBe(false));
    expect((await createTestClient().labels.retrieve(lbl.key)).name).toBe("Nominal");
  });

  it("should show a rename from another client and keep the color", async () => {
    const lbl = await createLabel();
    const { result } = await renderForm(lbl.key);
    const prevColor = result.current.form.get(path.color).value;
    await act(async () => {
      await client.labels.dispatch(
        lbl.key,
        label.set({ path: ["name"], value: "Anomaly" }),
      );
    });
    await waitFor(() => expect(result.current.form.value().name).toBe("Anomaly"));
    expect(result.current.form.get(path.color).value).toBe(prevColor);
  });
});
