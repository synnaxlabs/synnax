// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, group, ontology } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Ontology } from "@/ontology";
import { Symbol } from "@/schematic/symbol";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("Symbol queries", () => {
  let controller: AbortController;
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({
      client,
      excludeFluxStores: [Ontology.RESOURCES_FLUX_STORE_KEY],
    });
  });

  beforeEach(() => {
    controller = new AbortController();
  });

  afterEach(() => {
    controller.abort();
  });

  describe("useList", () => {
    it("should return a list of symbols for a given parent", async () => {
      const parent = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "test-symbols-parent",
      });
      const symbol1 = await client.workspaces.schematics.symbols.create({
        name: "symbol1",
        parent: group.ontologyID(parent.key),
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          scaleStroke: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });
      const symbol2 = await client.workspaces.schematics.symbols.create({
        name: "symbol2",
        parent: group.ontologyID(parent.key),
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          scaleStroke: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });

      const { result } = renderHook(
        () =>
          Symbol.useList({ initialQuery: { parent: group.ontologyID(parent.key) } }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parent: group.ontologyID(parent.key) });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(result.current.data).toHaveLength(2);
      expect(result.current.data).toContain(symbol1.key);
      expect(result.current.data).toContain(symbol2.key);

      const retrievedSymbol1 = result.current.getItem(symbol1.key);
      expect(retrievedSymbol1?.name).toBe("symbol1");
      const retrievedSymbol2 = result.current.getItem(symbol2.key);
      expect(retrievedSymbol2?.name).toBe("symbol2");
    });

    it("should filter symbols by search term", async () => {
      const uniqueId = Math.random().toString(36).substring(7);
      const parent = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: `test-symbols-search-${uniqueId}`,
      });
      await client.workspaces.schematics.symbols.create({
        name: "valve red",
        parent: group.ontologyID(parent.key),
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          scaleStroke: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });
      await client.workspaces.schematics.symbols.create({
        name: "pump blue",
        parent: group.ontologyID(parent.key),
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          scaleStroke: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });
      await client.workspaces.schematics.symbols.create({
        name: "valve purple",
        parent: group.ontologyID(parent.key),
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          scaleStroke: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });

      const { result } = renderHook(
        () =>
          Symbol.useList({
            initialQuery: {
              parent: group.ontologyID(parent.key),
              searchTerm: "valve",
            },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({
          parent: group.ontologyID(parent.key),
          searchTerm: "valve",
        });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      });

      const symbolItems = result.current.getItem(result.current.data);
      const names = symbolItems.map((s) => s.name);
      expect(names).toContain("valve red");
      expect(names).toContain("valve purple");
    });

    it("should update when a new symbol is added", async () => {
      const uniqueId = Math.random().toString(36).substring(7);
      const parent = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: `test-symbols-live-${uniqueId}`,
      });
      await client.workspaces.schematics.symbols.create({
        name: "initial-symbol",
        parent: group.ontologyID(parent.key),
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          scaleStroke: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });

      const { result } = renderHook(
        () =>
          Symbol.useList({
            initialQuery: { parent: group.ontologyID(parent.key) },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parent: group.ontologyID(parent.key) });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(1);
      });

      await act(async () => {
        await client.workspaces.schematics.symbols.create({
          name: "new-symbol",
          parent: group.ontologyID(parent.key),
          data: {
            svg: "<svg></svg>",
            states: [],
            handles: [],
            variant: "static",
            scale: 1,
            scaleStroke: false,
            previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
          },
        });
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(2);
      });

      const symbolItems = result.current.data.map((key) => result.current.getItem(key));
      const names = symbolItems.map((s) => s?.name).filter(Boolean);
      expect(names).toContain("initial-symbol");
      expect(names).toContain("new-symbol");
    });
  });

  describe("retrieve", () => {
    it("should retrieve a single symbol by key", async () => {
      const symbol = await client.workspaces.schematics.symbols.create({
        name: "retrieve-test",
        parent: ontology.ROOT_ID,
        data: {
          svg: "<svg>test</svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          scaleStroke: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });

      const { result } = renderHook(() => Symbol.useRetrieve({ key: symbol.key }), {
        wrapper,
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data?.name).toBe("retrieve-test");
        expect(result.current.data?.data.svg).toBe("<svg>test</svg>");
      });
    });
  });

  describe("useForm", () => {
    it("should create a new symbol", async () => {
      const parent = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "test-symbol-create",
      });

      const { result } = renderHook(() => Symbol.useForm({ query: {} }), { wrapper });

      await act(async () => {
        result.current.form.set("name", "created-symbol");
        result.current.form.set("data.svg", "<svg>created</svg>");
        result.current.form.set("parent", group.ontologyID(parent.key));
        result.current.save();
      });

      const key = await waitFor(async () => {
        expect(result.current.variant).toEqual("success");
        const key = result.current.form.get<string>("key", { optional: true })?.value;
        expect(key).toBeDefined();
        return key;
      });

      const retrieved = await client.workspaces.schematics.symbols.retrieve({
        key: key!,
      });
      expect(retrieved.name).toBe("created-symbol");
      expect(retrieved.data.svg).toBe("<svg>created</svg>");

      const children = await client.ontology.retrieveChildren(
        group.ontologyID(parent.key),
      );
      expect(children.length).toBe(1);
      expect(children[0].id.key).toBe(retrieved.key);
      expect(children[0].name).toBe("created-symbol");
    });

    it("should update an existing symbol", async () => {
      const parent = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "test-symbol-update",
      });
      const symbol = await client.workspaces.schematics.symbols.create({
        name: "original-name",
        parent: group.ontologyID(parent.key),
        data: {
          svg: "<svg>original</svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          scaleStroke: false,
          previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
        },
      });

      const { result } = renderHook(
        () =>
          Symbol.useForm({
            query: { key: symbol.key },
          }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.form.get("name").value).toBe("original-name");
        expect(result.current.form.get("parent").value).toEqual(
          group.ontologyID(parent.key),
        );
        expect(result.current.form.get("data.svg").value).toBe("<svg>original</svg>");
      });

      await act(async () => {
        result.current.form.set("name", "updated-name");
        result.current.form.set("data.svg", "<svg>updated</svg>");
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const retrieved = await client.workspaces.schematics.symbols.retrieve({
        key: symbol.key,
      });
      expect(retrieved.name).toBe("updated-name");
      expect(retrieved.data.svg).toBe("<svg>updated</svg>");
    });
  });

  describe("useRename", () => {
    it("should rename an existing symbol", async () => {
      const symbol = await client.workspaces.schematics.symbols.create({
        name: "original-name",
        parent: ontology.ROOT_ID,
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          scaleStroke: false,
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

      const retrieved = await client.workspaces.schematics.symbols.retrieve({
        key: symbol.key,
      });
      expect(retrieved.name).toBe("new-name");
    });
  });

  describe("useDelete", () => {
    it("should delete an existing symbol", async () => {
      const symbol = await client.workspaces.schematics.symbols.create({
        name: "to-be-deleted",
        parent: ontology.ROOT_ID,
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "static",
          scale: 1,
          scaleStroke: false,
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
        client.workspaces.schematics.symbols.retrieve({
          keys: [symbol.key],
        }),
      ).rejects.toThrow("not found");
    });
  });

  describe("useGroup", () => {
    it("should retrieve the symbol group", async () => {
      const { result } = renderHook(() => Symbol.useRetrieveGroup({ params: {} }), {
        wrapper,
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toBeDefined();
        expect(result.current.data?.name).toBe("Schematic Symbols");
      });
    });
  });
});
