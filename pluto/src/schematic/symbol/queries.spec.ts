// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
} from "@synnaxlabs/client/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Symbol } from "@/schematic/symbol";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const path = Symbol.formPath;

describe("Symbol queries", () => {
  let controller: AbortController;
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  beforeEach(() => {
    controller = new AbortController();
  });

  afterEach(() => {
    controller.abort();
  });

  describe("useList", () => {
    it("should list the symbols of a group by name", async () => {
      const parent = await client.groups.create({
        scope: "schematic_symbol",
        name: "test-symbols-parent",
      });
      const data: schematic.symbol.Symbol["data"] = {
        svg: "<svg></svg>",
        states: [],
        handles: [],
        variant: "static",
        scale: 1,
        strokeScaled: false,
        previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
      };
      const b = await client.schematics.symbols.create({
        name: "b",
        group: parent.key,
        data,
      });
      const a = await client.schematics.symbols.create({
        name: "a",
        group: parent.key,
        data,
      });
      const { result } = renderHook(
        () => Symbol.useList({ query: { groups: [parent.key] } }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual([a.key, b.key]);
      });
      await client.schematics.symbols.rename(a.key, "c");
      await waitFor(() => expect(result.current.data).toEqual([b.key, a.key]));
    });
  });

  describe("retrieve", () => {
    it("should retrieve a single symbol by key", async () => {
      const symbol = await client.schematics.symbols.create({
        name: "retrieve-test",
        data: {
          svg: "<svg>test</svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          strokeScaled: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });

      const { result } = await renderHookSuspended(
        () => Symbol.use({ key: symbol.key }),
        {
          wrapper,
        },
      );

      await waitFor(() => {
        expect(result.current).not.toBeNull();
        expect(result.current?.name).toBe("retrieve-test");
        expect(result.current?.data.svg).toBe("<svg>test</svg>");
      });
    });
  });

  describe("useForm", () => {
    it("should create a new symbol", async () => {
      const parent = await client.groups.create({
        scope: "schematic_symbol",
        name: "test-symbol-create",
      });

      const { result } = renderHook(() => Symbol.useForm({ query: null }), { wrapper });

      await act(async () => {
        result.current.form.set(path.name, "created-symbol");
        result.current.form.set(path.data.svg, "<svg>created</svg>");
        result.current.form.set(path.group, parent.key);
        result.current.save();
      });

      const key = await waitFor(async () => {
        expect(result.current.variant).toEqual("success");
        const key = result.current.form.get(path.key, { optional: true })?.value;
        expect(key).toBeDefined();
        return key;
      });

      const retrieved = await client.schematics.symbols.retrieve(key!);
      expect(retrieved.name).toBe("created-symbol");
      expect(retrieved.data.svg).toBe("<svg>created</svg>");
      expect(retrieved.group).toBe(parent.key);
    });

    it("should edit the symbol it created on a later save", async () => {
      const parent = await client.groups.create({
        scope: "schematic_symbol",
        name: "test-symbol-resave",
      });
      const { result } = renderHook(() => Symbol.useForm({ query: null }), { wrapper });
      act(() => {
        result.current.form.set(path.name, "created-symbol");
        result.current.form.set(path.data.svg, "<svg>created</svg>");
        result.current.form.set(path.group, parent.key);
      });
      await act(async () => expect(await result.current.saveAsync()).toBe(true));
      const key = result.current.form.get(path.key).value;
      if (key == null) throw new Error("symbol key is null");
      await createTestClient().schematics.symbols.dispatch(
        key,
        schematic.symbol.set({ path: ["data", "svg"], value: "<svg>remote</svg>" }),
      );
      act(() => result.current.form.set(path.name, "renamed-symbol"));
      await act(async () => expect(await result.current.saveAsync()).toBe(true));
      const stored = await createTestClient().schematics.symbols.retrieve(key);
      expect(stored.name).toBe("renamed-symbol");
      expect(stored.data.svg).toBe("<svg>remote</svg>");
    });

    it("should keep typed values and show the error on a refused save", async () => {
      const parent = await client.groups.create({
        scope: "schematic_symbol",
        name: "test-symbol-refused",
      });
      const symbol = await client.schematics.symbols.create({
        name: "refused-symbol",
        group: parent.key,
        data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
      });
      const viewer = await createTestClientWithRole(client, "Viewer");
      const { result } = await renderHookSuspended(
        () => Symbol.useForm({ query: { key: symbol.key } }),
        { wrapper: await createAsyncSynnaxWrapper({ client: viewer }) },
      );
      act(() => result.current.form.set(path.name, "typed-symbol"));
      await act(async () => expect(await result.current.saveAsync()).toBe(false));
      expect(result.current.variant).toEqual("error");
      expect(result.current.form.value().name).toEqual("typed-symbol");
    });

    it("should serve a warm symbol from the cache without suspending", async () => {
      const parent = await client.groups.create({
        scope: "schematic_symbol",
        name: "test-symbol-warm",
      });
      const symbol = await client.schematics.symbols.create({
        name: "warm-symbol",
        group: parent.key,
        data: {
          svg: "<svg>warm</svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          strokeScaled: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });
      // The cache answers only queries it maintains, so the read needs a live
      // subscriber before it is within reach of getCached.
      const stopSymbol = client.schematics.symbols.watch(symbol.key).onChange(() => {});
      try {
        await client.schematics.symbols.retrieve(symbol.key);
        // A plain renderHook resolves only if the read never suspends.
        const { result } = renderHook(
          () => Symbol.useForm({ query: { key: symbol.key } }),
          { wrapper },
        );
        expect(result.current.form.get(path.name).value).toBe("warm-symbol");
        expect(result.current.form.get(path.data.svg).value).toBe("<svg>warm</svg>");
        expect(result.current.form.get(path.group).value).toBe(parent.key);
      } finally {
        stopSymbol();
      }
    });

    it("should update an existing symbol", async () => {
      const parent = await client.groups.create({
        scope: "schematic_symbol",
        name: "test-symbol-update",
      });
      const symbol = await client.schematics.symbols.create({
        name: "original-name",
        group: parent.key,
        data: {
          svg: "<svg>original</svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          strokeScaled: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });

      const { result } = await renderHookSuspended(
        () =>
          Symbol.useForm({
            query: { key: symbol.key },
          }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.form.get(path.name).value).toBe("original-name");
        expect(result.current.form.get(path.group).value).toBe(parent.key);
        expect(result.current.form.get(path.data.svg).value).toBe(
          "<svg>original</svg>",
        );
      });

      const previewViewport = { zoom: 2, position: { x: 3, y: 4 } };
      await act(async () => {
        result.current.form.set(path.name, "updated-name");
        result.current.form.set(path.data.svg, "<svg>updated</svg>");
        result.current.form.set(path.data.strokeScaled, true);
        result.current.form.set(path.data.previewViewport, previewViewport);
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await waitFor(async () => {
        const retrieved = await client.schematics.symbols.retrieve(symbol.key);
        expect(retrieved.name).toBe("updated-name");
        expect(retrieved.group).toBe(parent.key);
        expect(retrieved.data.svg).toBe("<svg>updated</svg>");
        expect(retrieved.data.strokeScaled).toBe(true);
        expect(retrieved.data.previewViewport).toEqual(previewViewport);
      });
    });
  });

  describe("useRename", () => {
    it("should rename an existing symbol", async () => {
      const symbol = await client.schematics.symbols.create({
        name: "original-name",
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          strokeScaled: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });

      const { result } = renderHook(Symbol.useRename, { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: symbol.key, name: "new-name" });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const retrieved = await client.schematics.symbols.retrieve(symbol.key);
      expect(retrieved.name).toBe("new-name");
    });

    it("should apply the rename optimistically", async () => {
      const symbol = await client.schematics.symbols.create({
        name: "original-name",
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          strokeScaled: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });
      const { result } = renderHook(() => Symbol.useRename(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: symbol.key, name: "new-name" });
      });
      expect(client.schematics.symbols.watch(symbol.key).getCached()).toMatchObject({
        name: "new-name",
      });
    });
  });

  describe("useDelete", () => {
    it("should delete an existing symbol", async () => {
      const symbol = await client.schematics.symbols.create({
        name: "to-be-deleted",
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          strokeScaled: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });

      const { result } = renderHook(Symbol.useDelete, { wrapper });

      await act(async () => {
        await result.current.updateAsync(symbol.key);
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(
        client.schematics.symbols.retrieve({ key: symbol.key }),
      ).rejects.toThrow("not found");
    });
  });

  describe("useDeleteGroup", () => {
    const SYMBOL_DATA: schematic.symbol.New["data"] = {
      svg: "<svg></svg>",
      states: [],
      handles: [],
      variant: "static",
      scale: 1,
      strokeScaled: false,
      previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
    };

    const createPopulatedGroup = async () => {
      const grp = await client.groups.create({
        scope: "schematic_symbol",
        name: "group-to-be-deleted",
      });
      const symbol = await client.schematics.symbols.create({
        name: "symbol-in-group",
        group: grp.key,
        data: SYMBOL_DATA,
      });
      return { grp, symbol };
    };

    it("should delete the group and its symbols", async () => {
      const { grp, symbol } = await createPopulatedGroup();

      const { result } = renderHook(Symbol.useDeleteGroup, { wrapper });

      await act(async () => {
        await result.current.updateAsync(grp.key);
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(
        client.schematics.symbols.retrieve({ key: symbol.key }),
      ).rejects.toThrow("not found");
      await expect(client.groups.retrieve({ key: grp.key })).rejects.toThrow(
        "not found",
      );
    });

    it("should run afterOptimistic before the delete commits", async () => {
      const { grp } = await createPopulatedGroup();
      const order: string[] = [];

      const { result } = renderHook(
        () =>
          Symbol.useDeleteGroup({
            afterOptimistic: ({ data }) => {
              order.push(`optimistic:${data}`);
            },
            afterSuccess: () => {
              order.push("success");
            },
          }),
        { wrapper },
      );

      await act(async () => {
        await result.current.updateAsync(grp.key);
      });

      expect(order).toEqual([`optimistic:${grp.key}`, "success"]);
    });
  });
});
