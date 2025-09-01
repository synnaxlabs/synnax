// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, ontology } from "@synnaxlabs/client";
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
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-symbols-parent",
      );
      const symbol1 = await client.workspaces.schematic.symbols.create({
        name: "symbol1",
        parent: parent.ontologyID,
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
      const symbol2 = await client.workspaces.schematic.symbols.create({
        name: "symbol2",
        parent: parent.ontologyID,
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
        () => Symbol.useList({ initialParams: { parent: parent.ontologyID } }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parent: parent.ontologyID });
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
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        `test-symbols-search-${uniqueId}`,
      );
      await client.workspaces.schematic.symbols.create({
        name: "valve red",
        parent: parent.ontologyID,
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
      await client.workspaces.schematic.symbols.create({
        name: "pump blue",
        parent: parent.ontologyID,
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
      await client.workspaces.schematic.symbols.create({
        name: "valve purple",
        parent: parent.ontologyID,
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
            initialParams: { parent: parent.ontologyID, searchTerm: "valve" },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parent: parent.ontologyID, searchTerm: "valve" });
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
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        `test-symbols-live-${uniqueId}`,
      );
      await client.workspaces.schematic.symbols.create({
        name: "initial-symbol",
        parent: parent.ontologyID,
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
            initialParams: { parent: parent.ontologyID },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parent: parent.ontologyID });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(1);
      });

      await act(async () => {
        await client.workspaces.schematic.symbols.create({
          name: "new-symbol",
          parent: parent.ontologyID,
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
      const symbol = await client.workspaces.schematic.symbols.create({
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

      const { result } = renderHook(
        () => Symbol.retrieve.useDirect({ params: { key: symbol.key } }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data?.name).toBe("retrieve-test");
        expect(result.current.data?.data.svg).toBe("<svg>test</svg>");
      });
    });
  });

  describe("useForm", () => {
    it("should create a new symbol", async () => {
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-symbol-create",
      );

      const { result } = renderHook(
        () => Symbol.useForm({ params: { parent: parent.ontologyID } }),
        { wrapper },
      );

      await act(async () => {
        result.current.form.set("name", "created-symbol");
        result.current.form.set("data.svg", "<svg>created</svg>");
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const key = result.current.form.get<string>("key")?.value;
      expect(key).toBeDefined();

      const retrieved = await client.workspaces.schematic.symbols.retrieve({
        key,
      });
      expect(retrieved.name).toBe("created-symbol");
      expect(retrieved.data.svg).toBe("<svg>created</svg>");
    });

    it("should update an existing symbol", async () => {
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-symbol-update",
      );
      const symbol = await client.workspaces.schematic.symbols.create({
        name: "original-name",
        parent: parent.ontologyID,
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
            params: { key: symbol.key, parent: parent.ontologyID },
          }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.form.get("name").value).toBe("original-name");
      });

      await act(async () => {
        result.current.form.set("name", "updated-name");
        result.current.form.set("data.svg", "<svg>updated</svg>");
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const retrieved = await client.workspaces.schematic.symbols.retrieve({
        key: symbol.key,
      });
      expect(retrieved.name).toBe("updated-name");
      expect(retrieved.data.svg).toBe("<svg>updated</svg>");
    });
  });

  describe("useRename", () => {
    it("should rename an existing symbol", async () => {
      const symbol = await client.workspaces.schematic.symbols.create({
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

      const { result } = renderHook(
        () => Symbol.useRename({ params: { key: symbol.key } }),
        { wrapper },
      );

      await act(async () => {
        await result.current.updateAsync("new-name");
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const retrieved = await client.workspaces.schematic.symbols.retrieve({
        key: symbol.key,
      });
      expect(retrieved.name).toBe("new-name");
    });
  });

  describe("useDelete", () => {
    it("should delete an existing symbol", async () => {
      const symbol = await client.workspaces.schematic.symbols.create({
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

      const { result } = renderHook(
        () => Symbol.useDelete({ params: { key: symbol.key } }),
        { wrapper },
      );

      await act(async () => {
        await result.current.updateAsync();
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(
        client.workspaces.schematic.symbols.retrieve({
          keys: [symbol.key],
        }),
      ).rejects.toThrow("not found");
    });
  });

  describe("useGroup", () => {
    it("should retrieve the symbol group", async () => {
      const { result } = renderHook(
        () => Symbol.retrieveGroup.useDirect({ params: {} }),
        {
          wrapper,
        },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toBeDefined();
        expect(result.current.data?.name).toBe("Schematic Symbols");
      });
    });
  });
});
