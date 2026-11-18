// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type group, NotFoundError } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color as xcolor, compare, id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Color } from "@/color";
import { Flux } from "@/flux";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const remote = createTestClient();

const createNamed = async (): Promise<color.Named> =>
  await remote.colors.create({ name: `color-${id.create()}`, light: "#00FF00" });

const createPalette = async (): Promise<group.Group> =>
  await remote.groups.create({ scope: "color", name: `palette-${id.create()}` });

const createInPalette = async (palette: group.Group): Promise<color.Named> =>
  await remote.colors.create({
    name: `color-${id.create()}`,
    light: "#00FF00",
    group: palette.key,
  });

const renameRemotely = async (key: color.Key, name: string): Promise<void> => {
  await remote.colors.retrieve(key);
  await remote.colors.dispatch(key, color.set({ path: ["name"], value: name }));
};

describe("queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    await remote.connect();
  });

  describe("usePalettes", () => {
    it("should list each palette's colors in order, then loose colors under No palette", async () => {
      const palette = await createPalette();
      // Key order is the reverse of position order.
      const key = uuid.create().slice(1);
      const second = await remote.colors.create({
        key: `0${key}`,
        name: `color-${id.create()}`,
        light: "#00FF00",
        group: palette.key,
        position: "a1",
      });
      const first = await remote.colors.create({
        key: `f${key}`,
        name: `color-${id.create()}`,
        light: "#00FF00",
        group: palette.key,
        position: "a0",
      });
      const loose = await createNamed();
      const { result } = renderHook(() => Color.usePalettes(), { wrapper });
      await waitFor(() => {
        const found = result.current.find((p) => p.key === palette.key);
        expect(found?.colors.map((c) => c.key)).toEqual([first.key, second.key]);
        const last = result.current.at(-1);
        expect(last?.name).toBe("No palette");
        expect(last?.colors.map((c) => c.key)).toContain(loose.key);
      });
      const names = result.current.slice(0, -1).map((p) => p.name);
      expect(names).toEqual([...names].sort(compare.text));
    });
  });

  describe("useSeries", () => {
    it("should list only the series colors, in palette order", async () => {
      const loose = await createNamed();
      const series = (
        await remote.colors.retrieve({ groups: [color.SERIES_PALETTE] })
      ).sort(color.compare);
      const { result } = renderHook(
        () => ({ series: Color.useSeries(), all: Color.useResultList({}) }),
        { wrapper },
      );
      await waitFor(() => {
        const { all } = result.current;
        expect(
          all.variant === "success" && all.data.some((c) => c.key === loose.key),
        ).toBe(true);
        expect(result.current.series).toEqual(series.map((c) => c.key));
      });
    });
  });

  describe("useSeriesValues", () => {
    it("should resolve the series colors for the theme, in palette order", async () => {
      const series = (
        await remote.colors.retrieve({ groups: [color.SERIES_PALETTE] })
      ).sort(color.compare);
      const { result } = renderHook(
        () => ({ values: Color.useSeriesValues(), theme: Theming.use() }),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current.values).toEqual(
          series.map((c) => Theming.resolve(c, result.current.theme)),
        ),
      );
    });
  });

  describe("useResolve", () => {
    it("should resolve a fixed reference to its color", () => {
      const { result } = renderHook(() => Color.useResolve(), { wrapper });
      expect(
        result.current({ type: "fixed", value: xcolor.construct("#FF0000") }),
      ).toEqual(xcolor.construct("#FF0000"));
    });

    it("should resolve a named reference to its color, or undefined if missing", async () => {
      const named = await createNamed();
      const { result } = renderHook(() => Color.useResolve(), { wrapper });
      await waitFor(() =>
        expect(result.current({ type: "named", key: named.key })).toEqual(
          xcolor.construct(named.light),
        ),
      );
      expect(result.current({ type: "named", key: uuid.create() })).toBeUndefined();
    });
  });

  describe("useRetrieve", () => {
    it("should show a rename from another client", async () => {
      const named = await createNamed();
      const { result } = await renderHookSuspended(
        () => Color.useRetrieve({ key: named.key }),
        { wrapper },
      );
      expect(result.current).toEqual(named);
      await act(async () => await renameRemotely(named.key, "Anomaly"));
      await waitFor(() => expect(result.current.name).toBe("Anomaly"));
    });
  });

  describe("useRetrieveMultiple", () => {
    it("should show a rename from another client", async () => {
      const [a, b] = await Promise.all([createNamed(), createNamed()]);
      const { result } = await renderHookSuspended(
        () => Color.useRetrieveMultiple({ keys: [a.key, b.key] }),
        { wrapper },
      );
      expect(result.current).toEqual([a, b]);
      await act(async () => await renameRemotely(a.key, "Anomaly"));
      await waitFor(() =>
        expect(result.current.map((n) => n.name)).toEqual(["Anomaly", b.name]),
      );
    });
  });

  describe("list", () => {
    it("should show a rename from another client", async () => {
      const named = await createNamed();
      const { result } = renderHook(
        () =>
          Flux.useList((c) => c.colors, {
            name: "named colors",
            query: { where: { key: { in: [named.key] } } },
          }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toEqual([named.key]));
      await act(async () => await renameRemotely(named.key, "Anomaly"));
      await waitFor(() =>
        expect(result.current.getItem(named.key)?.name).toBe("Anomaly"),
      );
    });
  });

  describe("useCreate", () => {
    it("should create a named color", async () => {
      const { result } = renderHook(Color.useCreate, { wrapper });
      const name = `created-${id.create()}`;
      await act(async () => {
        await result.current.updateAsync({ name, light: "#FF0000" });
      });
      expect(result.current.variant).toEqual("success");
      expect((await createTestClient().colors.retrieve(name)).name).toBe(name);
    });
  });

  describe("useRename", () => {
    it("should rename a named color", async () => {
      const named = await createNamed();
      const { result } = renderHook(Color.useRename, { wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: named.key, name: "Anomaly" });
      });
      expect(result.current.variant).toEqual("success");
      const stored = await createTestClient().colors.retrieve(named.key);
      expect(stored.name).toBe("Anomaly");
    });

    it("should fail for an empty name", async () => {
      const named = await createNamed();
      const { result } = renderHook(Color.useRename, { wrapper });
      await act(async () => {
        expect(await result.current.updateAsync({ key: named.key, name: "" })).toBe(
          false,
        );
      });
      expect(result.current.variant).toEqual("error");
    });
  });

  describe("useDelete", () => {
    it("should delete a named color", async () => {
      const named = await createNamed();
      const { result } = renderHook(Color.useDelete, { wrapper });
      await act(async () => {
        await result.current.updateAsync(named.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(createTestClient().colors.retrieve(named.key)).rejects.toThrow();
    });
  });

  describe("useCreateForm", () => {
    it("should create a named color at its place in a palette", async () => {
      const palette = await createPalette();
      const { result } = renderHook(
        () =>
          Color.useCreateForm({
            query: null,
            initialValues: {
              name: "",
              light: xcolor.construct("#FF0000"),
              group: palette.key,
              position: "a0",
            },
          }),
        { wrapper },
      );
      const name = `created-${id.create()}`;
      act(() => {
        result.current.form.set(Color.createFormPath.name, name);
        result.current.save();
      });
      await waitFor(() => expect(result.current.form.value().key).toBeDefined());
      const stored = await createTestClient().colors.retrieve(name);
      expect(stored.group).toBe(palette.key);
      expect(stored.position).toBe("a0");
    });
  });

  describe("useForm", () => {
    it("should send an edit to the dark value", async () => {
      const named = await createNamed();
      const { result } = await renderHookSuspended(
        () => Color.useForm({ query: { key: named.key } }),
        { wrapper },
      );
      act(() =>
        result.current.form.set(Color.formPath.dark, xcolor.construct("#0000FF")),
      );
      await act(async () => expect(await result.current.flush()).toBe(true));
      const stored = await createTestClient().colors.retrieve(named.key);
      expect(stored.dark).toEqual(xcolor.construct("#0000FF"));
    });
  });

  describe("useMove", () => {
    it("should move named colors to new places in their palette", async () => {
      const [a, b] = await Promise.all([createNamed(), createNamed()]);
      const { result } = renderHook(Color.useMove, { wrapper });
      await act(async () => {
        await result.current.updateAsync([
          { key: a.key, position: "a5" },
          { key: b.key, position: "a4" },
        ]);
      });
      expect(result.current.variant).toEqual("success");
      const reader = createTestClient();
      expect((await reader.colors.retrieve(a.key)).position).toBe("a5");
      expect((await reader.colors.retrieve(b.key)).position).toBe("a4");
    });
  });

  describe("useRestore", () => {
    it("should write the shipped values back onto an edited built-in", async () => {
      const shipped = (await client.colors.retrieveBuiltins()).at(-1);
      if (shipped == null) throw new Error("the Core ships no named colors");
      await remote.colors.retrieve(shipped.key);
      await remote.colors.dispatch(
        shipped.key,
        color.set({ path: ["light"], value: xcolor.construct("#123456") }),
      );
      const { result } = renderHook(Color.useRestore, { wrapper });
      await act(async () => {
        await result.current.updateAsync(shipped);
      });
      expect(result.current.variant).toEqual("success");
      const stored = await createTestClient().colors.retrieve(shipped.key);
      expect(stored.light).toEqual(shipped.light);
      expect(stored.dark).toEqual(shipped.dark);
    });
  });

  describe("useDeletePalette", () => {
    it("should delete a palette and its named colors", async () => {
      const palette = await createPalette();
      const [a, b] = await Promise.all([
        createInPalette(palette),
        createInPalette(palette),
      ]);
      const { result } = renderHook(Color.useDeletePalette, { wrapper });
      await act(async () => {
        await result.current.updateAsync({
          palette: palette.key,
          colors: [a.key, b.key],
        });
      });
      expect(result.current.variant).toEqual("success");
      const reader = createTestClient();
      await expect(reader.groups.retrieve({ key: palette.key })).rejects.toThrow(
        NotFoundError,
      );
      await expect(reader.colors.retrieve(a.key)).rejects.toThrow(NotFoundError);
      await expect(reader.colors.retrieve(b.key)).rejects.toThrow(NotFoundError);
    });

    it("should delete an empty palette", async () => {
      const palette = await createPalette();
      const { result } = renderHook(Color.useDeletePalette, { wrapper });
      await act(async () => {
        await result.current.updateAsync({ palette: palette.key, colors: [] });
      });
      expect(result.current.variant).toEqual("success");
      await expect(
        createTestClient().groups.retrieve({ key: palette.key }),
      ).rejects.toThrow(NotFoundError);
    });
  });
});
