// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { enumeration } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Enumeration } from "@/enumeration";
import { Flux } from "@/flux";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const remote = createTestClient();

const createEnumeration = async (): Promise<enumeration.Enumeration> =>
  await remote.enumerations.create({
    name: `enumeration-${id.create()}`,
    entries: [
      { key: "off", value: 0, name: "Off" },
      { key: "on", value: 1, name: "On" },
    ],
  });

const renameRemotely = async (key: enumeration.Key, name: string): Promise<void> => {
  await remote.enumerations.retrieve(key);
  await remote.enumerations.dispatch(
    key,
    enumeration.set({ path: ["name"], value: name }),
  );
};

describe("queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    await remote.connect();
  });

  describe("useRetrieve", () => {
    it("should show an entry edit from another client", async () => {
      const created = await createEnumeration();
      const { result } = await renderHookSuspended(
        () => Enumeration.useRetrieve({ key: created.key }),
        { wrapper },
      );
      expect(result.current).toEqual(created);
      await act(async () => {
        await remote.enumerations.retrieve(created.key);
        await remote.enumerations.dispatch(
          created.key,
          enumeration.set({ path: ["entries", "on", "name"], value: "Running" }),
        );
      });
      await waitFor(() => expect(result.current.entries[1].name).toBe("Running"));
    });
  });

  describe("useRetrieveMultiple", () => {
    it("should show a rename from another client", async () => {
      const [a, b] = await Promise.all([createEnumeration(), createEnumeration()]);
      const { result } = await renderHookSuspended(
        () => Enumeration.useRetrieveMultiple({ keys: [a.key, b.key] }),
        { wrapper },
      );
      expect(result.current).toEqual([a, b]);
      await act(async () => await renameRemotely(a.key, "Breaker"));
      await waitFor(() =>
        expect(result.current.map((e) => e.name)).toEqual(["Breaker", b.name]),
      );
    });
  });

  describe("useResultList", () => {
    it("should list the matched enumerations and show a rename from another client", async () => {
      const [a, b] = await Promise.all([createEnumeration(), createEnumeration()]);
      const { result } = renderHook(
        () => Enumeration.useResultList({ where: { key: { in: [a.key, b.key] } } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toBe("success"));
      expect(result.current.data?.map((e) => e.key).sort()).toEqual(
        [a.key, b.key].sort(),
      );
      await act(async () => await renameRemotely(a.key, "Breaker"));
      await waitFor(() =>
        expect(result.current.data?.find((e) => e.key === a.key)?.name).toBe("Breaker"),
      );
    });
  });

  describe("list", () => {
    it("should show a rename from another client", async () => {
      const created = await createEnumeration();
      const { result } = renderHook(
        () =>
          Flux.useList((c) => c.enumerations, {
            name: "enumerations",
            query: { where: { key: { in: [created.key] } } },
          }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toEqual([created.key]));
      await act(async () => await renameRemotely(created.key, "Breaker"));
      await waitFor(() =>
        expect(result.current.getItem(created.key)?.name).toBe("Breaker"),
      );
    });
  });

  describe("useResultBuiltins", () => {
    it("should return the shipped built-in enumerations", async () => {
      const { result } = renderHook(() => Enumeration.useResultBuiltins({}), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toBe("success"));
      expect(result.current.data?.map((e) => e.name)).toEqual([
        "Valve",
        "Normally open valve",
        "Pump",
        "Switch",
        "Light",
      ]);
    });
  });

  describe("useResult", () => {
    it("should return empty uses of an unused enumeration", async () => {
      const created = await createEnumeration();
      const { result } = renderHook(
        () => Enumeration.useResult({ key: created.key, includeUses: true }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toBe("success"));
      expect(result.current.data?.uses).toEqual([]);
    });
  });

  describe("useCreate", () => {
    it("should create an enumeration", async () => {
      const { result } = renderHook(Enumeration.useCreate, { wrapper });
      const name = `created-${id.create()}`;
      await act(async () => {
        await result.current.updateAsync({
          name,
          entries: [{ key: "off", value: 0, name: "Off" }],
        });
      });
      expect(result.current.variant).toEqual("success");
      const stored = await createTestClient().enumerations.retrieve(name);
      expect(stored.entries.map((e) => e.name)).toEqual(["Off"]);
    });
  });

  describe("useRename", () => {
    it("should rename an enumeration", async () => {
      const created = await createEnumeration();
      const { result } = renderHook(Enumeration.useRename, { wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: created.key, name: "Breaker" });
      });
      expect(result.current.variant).toEqual("success");
      const stored = await createTestClient().enumerations.retrieve(created.key);
      expect(stored.name).toBe("Breaker");
    });

    it("should fail for an empty name", async () => {
      const created = await createEnumeration();
      const { result } = renderHook(Enumeration.useRename, { wrapper });
      await act(async () => {
        expect(await result.current.updateAsync({ key: created.key, name: "" })).toBe(
          false,
        );
      });
      expect(result.current.variant).toEqual("error");
    });
  });

  describe("useDelete", () => {
    it("should delete an enumeration", async () => {
      const created = await createEnumeration();
      const { result } = renderHook(Enumeration.useDelete, { wrapper });
      await act(async () => {
        await result.current.updateAsync(created.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(
        createTestClient().enumerations.retrieve(created.key),
      ).rejects.toThrow();
    });
  });

  describe("useReplace", () => {
    it("should delete the replaced enumeration and keep the other", async () => {
      const [from, to] = await Promise.all([createEnumeration(), createEnumeration()]);
      const { result } = renderHook(Enumeration.useReplace, { wrapper });
      await act(async () => {
        await result.current.updateAsync({ from: from.key, to: to.key });
      });
      expect(result.current.variant).toEqual("success");
      const other = createTestClient();
      await expect(other.enumerations.retrieve(from.key)).rejects.toThrow();
      expect((await other.enumerations.retrieve(to.key)).key).toEqual(to.key);
    });
  });

  describe("useForm", () => {
    it("should send an entry rename", async () => {
      const created = await createEnumeration();
      const { result } = await renderHookSuspended(
        () => Enumeration.useForm({ query: { key: created.key } }),
        { wrapper },
      );
      act(() =>
        result.current.form.set(Enumeration.formPath.entries.on.name, "Closed"),
      );
      await act(async () => expect(await result.current.flush()).toBe(true));
      const stored = await createTestClient().enumerations.retrieve(created.key);
      expect(stored.entries.map((e) => e.name)).toEqual(["Off", "Closed"]);
    });

    it("should insert, move, and remove entries", async () => {
      const created = await createEnumeration();
      const { result } = await renderHookSuspended(
        () => Enumeration.useForm({ query: { key: created.key } }),
        { wrapper },
      );
      const { entries } = Enumeration.formPath;
      act(() =>
        result.current.form.insert(
          entries,
          { key: "tripped", value: 2, name: "Tripped", abnormal: true },
          "on",
        ),
      );
      act(() => result.current.form.move(entries, "on"));
      act(() => result.current.form.remove(entries.off));
      await act(async () => expect(await result.current.flush()).toBe(true));
      const stored = await createTestClient().enumerations.retrieve(created.key);
      expect(stored.entries.map((e) => e.key)).toEqual(["on", "tripped"]);
      expect(stored.entries[1].abnormal).toBe(true);
    });
  });

  describe("useRestore", () => {
    it("should write the shipped entries back onto an edited built-in", async () => {
      const shipped = (await client.enumerations.retrieveBuiltins()).at(-1);
      if (shipped == null) throw new Error("the Core ships no enumerations");
      await remote.enumerations.retrieve(shipped.key);
      await remote.enumerations.dispatch(shipped.key, [
        enumeration.set({ path: ["name"], value: `edited-${id.create()}` }),
        enumeration.set({ path: ["entries"], value: [] }),
      ]);
      const { result } = renderHook(Enumeration.useRestore, { wrapper });
      await act(async () => {
        await result.current.updateAsync(shipped);
      });
      expect(result.current.variant).toEqual("success");
      const stored = await createTestClient().enumerations.retrieve(shipped.key);
      expect(stored.name).toBe(shipped.name);
      expect(stored.entries).toEqual(shipped.entries);
    });
  });
});
