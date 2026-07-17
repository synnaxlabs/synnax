// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor, within } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { Errors } from "@/errors";
import { Panel } from "@/panel";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
// writer is a second connected client used to emit changes the wrapper client
// must pick up through the action channel.
const writer = createTestClient();

const newTab = (): panel.Tab => ({
  variant: "view",
  key: uuid.create(),
  type: "selector",
  args: {},
});

const asView = (tab?: panel.Tab | null): panel.TabView | undefined =>
  tab?.variant === "view" ? tab : undefined;

describe("Panel queries", () => {
  let controller: AbortController;
  let wrapper: FC<PropsWithChildren>;

  beforeEach(async () => {
    controller = new AbortController();
    wrapper = await createAsyncSynnaxWrapper({ client });
  });
  afterEach(() => {
    controller.abort();
  });

  const createPanel = async (): Promise<panel.Panel> =>
    await client.panels.create({ name: `panel-${uuid.create()}` });

  const asSplit = (node?: panel.Node): panel.NodeSplit | undefined =>
    node?.variant === "split" ? node : undefined;

  const leafTabKeys = (node?: panel.Node): string[] | undefined =>
    node?.variant === "leaf" ? node.tabs.map((t) => t.key) : undefined;

  const loadAndUse = async <T,>(key: panel.Key, hook: () => T) => {
    const retrieve = renderHook(() => Panel.useRetrieve({ key }), { wrapper });
    await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));
    return renderHook(hook, { wrapper });
  };

  const loadAndCount = async <T,>(key: panel.Key, hook: () => T) => {
    const retrieve = renderHook(() => Panel.useRetrieve({ key }), { wrapper });
    await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));
    let renderCount = 0;
    const { result } = renderHook(
      () => {
        renderCount++;
        return hook();
      },
      { wrapper },
    );
    return { result, renderCount: () => renderCount };
  };

  // createSplitPanel builds a panel whose root is a split: first (node 2) holds tabB,
  // last (node 3) holds tabA.
  const createSplitPanel = async () => {
    const created = await createPanel();
    const ops = await loadAndUse(created.key, () => Panel.useDispatch());
    const [tabA, tabB] = [newTab(), newTab()];
    await act(async () => {
      await ops.result.current.dispatchAsync({
        key: created.key,
        actions: [
          panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY }),
          panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_NODE_KEY }),
          panel.splitTab({ key: tabA.key, direction: "x" }),
        ],
      });
    });
    return { created, ops, tabA, tabB };
  };

  describe("useRetrieve", () => {
    it("should fetch a panel by key", async () => {
      const created = await client.panels.create({ name: "retrieve-target" });
      const { result } = renderHook(() => Panel.useRetrieve({ key: created.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.key).toEqual(created.key);
      expect(result.current.data?.name).toEqual("retrieve-target");
    });

    it("should cache retrieved panels", async () => {
      const created = await createPanel();
      const { result: first } = renderHook(
        () => Panel.useRetrieve({ key: created.key }),
        { wrapper },
      );
      await waitFor(() => expect(first.current.variant).toEqual("success"));

      const { result: second } = renderHook(
        () => Panel.useRetrieve({ key: created.key }),
        { wrapper },
      );
      await waitFor(() => expect(second.current.variant).toEqual("success"));
      expect(second.current.data).toEqual(first.current.data);
    });
  });

  describe("useEnsureRetrieved", () => {
    // Single-hook bootstrap component so the suspending useEnsureRetrieved is
    // not followed by additional hooks; that shape trips a React 19
    // concurrent-replay warning.
    it("populates the cache so downstream selectors resolve", async () => {
      const created = await createPanel();
      const Bootstrap = (): ReactElement => {
        Panel.useEnsureRetrieved({ key: created.key });
        return <div data-testid="loaded" />;
      };
      const Wrapper = wrapper;
      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary loading={null}>
              <Bootstrap />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });
      await within(utils.container).findByTestId("loaded");

      const { result } = renderHook(() => Panel.useSelectRoot({ key: created.key }), {
        wrapper,
      });
      expect(result.current.variant).toEqual("leaf");
    });
  });

  describe("useCreate", () => {
    it("should create a panel with a defaulted empty root", async () => {
      const { result } = renderHook(() => Panel.useCreate(), { wrapper });
      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({ key, name: "created-panel" });
      });
      expect(result.current.variant).toEqual("success");
      expect(result.current.data?.name).toEqual("created-panel");

      const { result: retrieved } = renderHook(() => Panel.useRetrieve({ key }), {
        wrapper,
      });
      await waitFor(() => expect(retrieved.current.variant).toEqual("success"));
      expect(retrieved.current.data?.name).toEqual("created-panel");
      expect(retrieved.current.data?.root).toEqual({ variant: "leaf", tabs: [] });
    });

    it("should cache the created panel", async () => {
      const { result } = renderHook(() => Panel.useCreate(), { wrapper });
      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({ key, name: "stored-panel" });
      });

      const { result: root } = await loadAndUse(key, () =>
        Panel.useSelectRoot({ key }),
      );
      expect(root.current).toEqual({ variant: "leaf", tabs: [] });
    });

    it("should parent the panel under the given resource", async () => {
      const parentProject = await client.projects.create({
        name: `parent-${uuid.create()}`,
        layout: {},
      });
      const parent = project.ontologyID(parentProject.key);

      const { result } = renderHook(() => Panel.useCreate(), { wrapper });
      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({ key, name: "parented-panel", parent });
      });

      const children = await client.ontology.retrieveChildren(parent);
      expect(children.map((c) => c.id.key)).toContain(key);
    });
  });

  describe("useList", () => {
    it("should return panels including those created beforehand", async () => {
      const p1 = await client.panels.create({ name: "list-a" });
      const p2 = await client.panels.create({ name: "list-b" });

      const { result } = renderHook(() => Panel.useList(), { wrapper });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(p1.key);
      expect(result.current.data).toContain(p2.key);
    });

    it("should expose individual panels via getItem", async () => {
      const target = await client.panels.create({ name: "get-item-target" });
      const { result } = renderHook(() => Panel.useList(), { wrapper });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const item = result.current.getItem(target.key);
      expect(item?.key).toEqual(target.key);
      expect(item?.name).toEqual("get-item-target");
    });
  });

  describe("useRename", () => {
    it("should rename an existing panel", async () => {
      const target = await client.panels.create({ name: "before-rename" });
      const { result } = renderHook(() => Panel.useRename(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: target.key, name: "after-rename" });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const fetched = await client.panels.retrieve(target.key);
      expect(fetched.name).toEqual("after-rename");
    });
  });

  describe("useDelete", () => {
    it("should delete an existing panel", async () => {
      const target = await client.panels.create({ name: "to-delete" });
      const { result } = renderHook(() => Panel.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(target.key);
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      await expect(client.panels.retrieve(target.key)).rejects.toThrow();
    });

    it("should delete multiple panels at once", async () => {
      const a = await createPanel();
      const b = await createPanel();
      const { result } = renderHook(() => Panel.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync([a.key, b.key]);
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      await expect(client.panels.retrieve(a.key)).rejects.toThrow();
      await expect(client.panels.retrieve(b.key)).rejects.toThrow();
    });
  });

  describe("useDispatch", () => {
    it("applies insert_tab and persists it to the server", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: Panel.useRetrieve({ key: created.key }),
        dispatch: Panel.useDispatch(),
      }));
      const tab = newTab();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.insertTab({ tab, targetLeaf: panel.ROOT_NODE_KEY })],
        });
      });
      await waitFor(() =>
        expect(leafTabKeys(result.current.retrieve.data?.root)).toEqual([tab.key]),
      );

      const fresh = await client.panels.retrieve(created.key);
      expect(fresh.root).toEqual(result.current.retrieve.data?.root);
    });

    it("splits the target leaf when insert_tab carries an edge location", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: Panel.useRetrieve({ key: created.key }),
        dispatch: Panel.useDispatch(),
      }));
      const [tabA, tabB] = [newTab(), newTab()];
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY })],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.insertTab({
              tab: tabB,
              targetLeaf: panel.ROOT_NODE_KEY,
              location: "right",
            }),
          ],
        });
      });
      await waitFor(() => {
        const root = asSplit(result.current.retrieve.data?.root);
        expect(root?.direction).toEqual("x");
        expect(leafTabKeys(root?.first)).toEqual([tabA.key]);
        expect(leafTabKeys(root?.last)).toEqual([tabB.key]);
      });

      const fresh = await client.panels.retrieve(created.key);
      expect(fresh.root).toEqual(result.current.retrieve.data?.root);
    });

    it("resizes a split and round-trips the new ratio", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: Panel.useRetrieve({ key: created.key }),
        dispatch: Panel.useDispatch(),
      }));
      const [tabA, tabB] = [newTab(), newTab()];
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.splitTab({ key: tabA.key, direction: "x" }),
          ],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.resizeSplit({ split: panel.ROOT_NODE_KEY, size: 0.25 })],
        });
      });
      await waitFor(() =>
        expect(asSplit(result.current.retrieve.data?.root)?.size).toEqual(0.25),
      );

      const fresh = await client.panels.retrieve(created.key);
      expect(asSplit(fresh.root)?.size).toEqual(0.25);
    });

    it("splits a tab off into a new sibling pane", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: Panel.useRetrieve({ key: created.key }),
        dispatch: Panel.useDispatch(),
      }));
      const [tabA, tabB] = [newTab(), newTab()];
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_NODE_KEY }),
          ],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.splitTab({ key: tabA.key, direction: "x" })],
        });
      });
      await waitFor(() => {
        const root = asSplit(result.current.retrieve.data?.root);
        expect(root?.direction).toEqual("x");
        expect(leafTabKeys(root?.first)).toEqual([tabB.key]);
        expect(leafTabKeys(root?.last)).toEqual([tabA.key]);
      });

      const fresh = await client.panels.retrieve(created.key);
      expect(fresh.root).toEqual(result.current.retrieve.data?.root);
    });

    it("ignores moving a leaf's only tab to an edge of its own leaf", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: Panel.useRetrieve({ key: created.key }),
        dispatch: Panel.useDispatch(),
      }));
      const tab = newTab();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.insertTab({ tab, targetLeaf: panel.ROOT_NODE_KEY })],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.moveTab({
              key: tab.key,
              targetLeaf: panel.ROOT_NODE_KEY,
              location: "right",
            }),
          ],
        });
      });
      await waitFor(() => {
        const root = result.current.retrieve.data?.root;
        expect(root?.variant).toEqual("leaf");
        expect(leafTabKeys(root)).toEqual([tab.key]);
      });

      const fresh = await client.panels.retrieve(created.key);
      expect(fresh.root).toEqual(result.current.retrieve.data?.root);
    });

    it("collapses the emptied leaf after remove_tab", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: Panel.useRetrieve({ key: created.key }),
        dispatch: Panel.useDispatch(),
      }));
      const [tabA, tabB] = [newTab(), newTab()];
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_NODE_KEY }),
          ],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.splitTab({ key: tabA.key, direction: "x" })],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.removeTab({ key: tabA.key })],
        });
      });
      await waitFor(() => {
        const root = result.current.retrieve.data?.root;
        expect(root?.variant).toEqual("leaf");
        expect(leafTabKeys(root)).toEqual([tabB.key]);
      });

      const fresh = await client.panels.retrieve(created.key);
      expect(fresh.root).toEqual(result.current.retrieve.data?.root);
    });

    it("set_tab_view replaces a tab's view content and persists it", async () => {
      const created = await createPanel();
      const tab = newTab();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: Panel.useRetrieve({ key: created.key }),
        dispatch: Panel.useDispatch(),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.insertTab({ tab, targetLeaf: panel.ROOT_NODE_KEY })],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.setTabView({
              key: tab.key,
              view: { type: "docs", args: { path: "/intro" } },
            }),
          ],
        });
      });
      const updated = asView(
        panel.findTab(result.current.retrieve.data!.root, tab.key),
      );
      expect(updated?.type).toEqual("docs");
      expect(updated?.args).toEqual({ path: "/intro" });

      const fresh = await client.panels.retrieve(created.key);
      const freshTab = asView(panel.findTab(fresh.root, tab.key));
      expect(freshTab?.type).toEqual("docs");
      expect(freshTab?.args).toEqual({ path: "/intro" });
    });

    it("set_tab_resource swaps a view tab to a resource and persists it", async () => {
      const created = await createPanel();
      const tab = newTab();
      const resource = { type: "lineplot", key: uuid.create() } as const;
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: Panel.useRetrieve({ key: created.key }),
        dispatch: Panel.useDispatch(),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.insertTab({ tab, targetLeaf: panel.ROOT_NODE_KEY })],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.setTabResource({ key: tab.key, resource })],
        });
      });
      const updated = panel.findTab(result.current.retrieve.data!.root, tab.key);
      expect(updated).toEqual({ variant: "resource", key: tab.key, resource });

      const fresh = await client.panels.retrieve(created.key);
      expect(panel.findTab(fresh.root, tab.key)).toEqual(updated);
    });
  });

  // Panel inverses are empty in Phase 1, so undo cannot restore document state
  // yet. The stack structure is still real: these tests count undo steps via
  // canUndo/canRedo to pin the gesture-coalescing contract.
  describe("undo stack", () => {
    it("coalesces a resize drag stream into a single undo step", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        dispatch: Panel.useDispatch(),
        undo: Panel.useUndo({ key: created.key }),
      }));
      const [tabA, tabB] = [newTab(), newTab()];
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.splitTab({ key: tabA.key, direction: "x" }),
          ],
        });
      });
      for (const size of [0.2, 0.3, 0.4])
        await act(async () => {
          await result.current.dispatch.dispatchAsync({
            key: created.key,
            actions: [panel.resizeSplit({ split: panel.ROOT_NODE_KEY, size })],
          });
        });
      expect(result.current.undo.canUndo).toBe(true);

      await act(async () => result.current.undo.undo());
      await waitFor(() => expect(result.current.undo.canUndo).toBe(true));
      await act(async () => result.current.undo.undo());
      await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
    });

    it("does not coalesce resizes of different splits", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        dispatch: Panel.useDispatch(),
        undo: Panel.useUndo({ key: created.key }),
      }));
      const [tabA, tabB, tabC] = [newTab(), newTab(), newTab()];
      const firstChild = panel.childNodeKey(panel.ROOT_NODE_KEY, "first");
      const dispatchOne = async (action: panel.Action) =>
        await act(async () => {
          await result.current.dispatch.dispatchAsync({
            key: created.key,
            actions: [action],
          });
        });
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.insertTab({ tab: tabC, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.splitTab({ key: tabA.key, direction: "x" }),
          ],
        });
      });
      await dispatchOne(panel.splitTab({ key: tabB.key, direction: "x" }));
      await dispatchOne(panel.resizeSplit({ split: panel.ROOT_NODE_KEY, size: 0.3 }));
      await dispatchOne(panel.resizeSplit({ split: firstChild, size: 0.6 }));

      for (let i = 0; i < 3; i++) {
        await act(async () => result.current.undo.undo());
        await waitFor(() => expect(result.current.undo.canUndo).toBe(true));
      }
      await act(async () => result.current.undo.undo());
      await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
    });

    it("coalesces a cross-leaf move_tab stream into a single undo step", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        dispatch: Panel.useDispatch(),
        undo: Panel.useUndo({ key: created.key }),
      }));
      const [tabA, tabB, tabC] = [newTab(), newTab(), newTab()];
      const firstChild = panel.childNodeKey(panel.ROOT_NODE_KEY, "first");
      const lastChild = panel.childNodeKey(panel.ROOT_NODE_KEY, "last");
      const dispatchOne = async (action: panel.Action) =>
        await act(async () => {
          await result.current.dispatch.dispatchAsync({
            key: created.key,
            actions: [action],
          });
        });
      await dispatchOne(
        panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY }),
      );
      await dispatchOne(
        panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_NODE_KEY }),
      );
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.insertTab({ tab: tabC, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.splitTab({ key: tabC.key, direction: "x" }),
          ],
        });
      });
      await dispatchOne(panel.moveTab({ key: tabA.key, targetLeaf: lastChild }));
      await dispatchOne(panel.moveTab({ key: tabA.key, targetLeaf: firstChild }));

      for (let i = 0; i < 3; i++) {
        await act(async () => result.current.undo.undo());
        await waitFor(() => expect(result.current.undo.canUndo).toBe(true));
      }
      await act(async () => result.current.undo.undo());
      await waitFor(() => expect(result.current.undo.canUndo).toBe(false));
    });

    it("transitions undo to redo and clears redo on a new dispatch", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        dispatch: Panel.useDispatch(),
        undo: Panel.useUndo({ key: created.key }),
        redo: Panel.useRedo({ key: created.key }),
      }));
      const [tabA, tabB] = [newTab(), newTab()];
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.splitTab({ key: tabA.key, direction: "x" }),
          ],
        });
      });
      expect(result.current.undo.canUndo).toBe(true);
      expect(result.current.redo.canRedo).toBe(false);

      await act(async () => result.current.undo.undo());
      await waitFor(() => {
        expect(result.current.undo.canUndo).toBe(false);
        expect(result.current.redo.canRedo).toBe(true);
      });

      await act(async () => result.current.redo.redo());
      await waitFor(() => {
        expect(result.current.undo.canUndo).toBe(true);
        expect(result.current.redo.canRedo).toBe(false);
      });

      await act(async () => result.current.undo.undo());
      await waitFor(() => expect(result.current.redo.canRedo).toBe(true));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.resizeSplit({ split: panel.ROOT_NODE_KEY, size: 0.7 })],
        });
      });
      await waitFor(() => expect(result.current.redo.canRedo).toBe(false));
    });
  });

  describe("selectors", () => {
    it("useSelectRoot returns the stored tree and updates after a split", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        root: Panel.useSelectRoot({ key: created.key }),
        dispatch: Panel.useDispatch(),
      }));
      expect(result.current.root.variant).toEqual("leaf");

      const [tabA, tabB] = [newTab(), newTab()];
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.splitTab({ key: tabA.key, direction: "x" }),
          ],
        });
      });
      await waitFor(() => expect(result.current.root.variant).toEqual("split"));
    });

    it("useSelectRoot keeps a stable reference across a rename", async () => {
      const created = await createPanel();
      const ops = await loadAndUse(created.key, () => ({
        retrieve: Panel.useRetrieve({ key: created.key }),
        rename: Panel.useRename(),
      }));
      const { result, renderCount } = await loadAndCount(created.key, () =>
        Panel.useSelectRoot({ key: created.key }),
      );
      const firstRoot = result.current;
      const countBefore = renderCount();

      await act(async () => {
        await ops.result.current.rename.updateAsync({
          key: created.key,
          name: "root-stable",
        });
      });
      await waitFor(() =>
        expect(ops.result.current.retrieve.data?.name).toEqual("root-stable"),
      );

      expect(result.current).toBe(firstRoot);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectTab returns the tab and updates after set_tab_view", async () => {
      const created = await createPanel();
      const tab = newTab();
      const ops = await loadAndUse(created.key, () => ({
        dispatch: Panel.useDispatch(),
      }));
      await act(async () => {
        await ops.result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.insertTab({ tab, targetLeaf: panel.ROOT_NODE_KEY })],
        });
      });

      const { result } = renderHook(
        () => Panel.useSelectTab({ key: created.key, tabKey: tab.key }),
        { wrapper },
      );
      expect(result.current.key).toEqual(tab.key);
      expect(asView(result.current)?.type).toEqual("selector");

      await act(async () => {
        await ops.result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.setTabView({
              key: tab.key,
              view: { type: "docs", args: { path: "/x" } },
            }),
          ],
        });
      });
      await waitFor(() => expect(asView(result.current)?.type).toEqual("docs"));
    });

    it("useSelectTab keeps a stable reference when a different tab changes", async () => {
      const created = await createPanel();
      const [tabA, tabB] = [newTab(), newTab()];
      const ops = await loadAndUse(created.key, () => ({
        dispatch: Panel.useDispatch(),
      }));
      await act(async () => {
        await ops.result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_NODE_KEY }),
          ],
        });
      });

      const { result, renderCount } = await loadAndCount(created.key, () =>
        Panel.useSelectTab({ key: created.key, tabKey: tabA.key }),
      );
      const firstTab = result.current;
      const countBefore = renderCount();

      await act(async () => {
        await ops.result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.setTabView({ key: tabB.key, view: { type: "docs" } })],
        });
      });

      expect(result.current).toBe(firstTab);
      expect(renderCount()).toEqual(countBefore);
    });
  });

  describe("useSelectSelection", () => {
    const createTabs = async (...tabs: panel.Tab[]) => {
      const created = await createPanel();
      const ops = await loadAndUse(created.key, () => Panel.useDispatch());
      await act(async () => {
        await ops.result.current.dispatchAsync({
          key: created.key,
          actions: tabs.map((tab) =>
            panel.insertTab({ tab, targetLeaf: panel.ROOT_NODE_KEY }),
          ),
        });
      });
      return { created, ops };
    };

    it("should keep only the most recent selected tab per leaf", async () => {
      const [tabA, tabB] = [newTab(), newTab()];
      const { created } = await createTabs(tabA, tabB);
      const { result } = renderHook(
        () =>
          Panel.useSelectSelection({
            key: created.key,
            selected: [tabB.key, tabA.key],
          }),
        { wrapper },
      );
      expect(result.current).toEqual([tabB.key]);
    });

    it("should drop keys no longer in the tree and add per-leaf fallbacks", async () => {
      const [tabA, tabB] = [newTab(), newTab()];
      const { created } = await createTabs(tabA, tabB);
      const removed = newTab();
      const { result } = renderHook(
        () => Panel.useSelectSelection({ key: created.key, selected: [removed.key] }),
        { wrapper },
      );
      expect(result.current).toEqual([tabA.key]);
    });

    it("should keep one recency-ordered tab per leaf across a split", async () => {
      const [tabA, tabB] = [newTab(), newTab()];
      const { created, ops } = await createTabs(tabA, tabB);
      await act(async () => {
        await ops.result.current.dispatchAsync({
          key: created.key,
          actions: [panel.splitTab({ key: tabB.key, direction: "x" })],
        });
      });
      const { result } = renderHook(
        () =>
          Panel.useSelectSelection({
            key: created.key,
            selected: [tabB.key, tabA.key],
          }),
        { wrapper },
      );
      expect(result.current).toEqual([tabB.key, tabA.key]);
    });

    it("should return the list unresolved when the panel is not cached", () => {
      const orphan = uuid.create();
      const { result } = renderHook(
        () => Panel.useSelectSelection({ key: orphan, selected: ["x", "y"] }),
        { wrapper },
      );
      expect(result.current).toEqual(["x", "y"]);
    });

    it("should stay referentially stable across a content-only change", async () => {
      const [tabA, tabB] = [newTab(), newTab()];
      const { created, ops } = await createTabs(tabA, tabB);
      const { result, renderCount } = await loadAndCount(created.key, () =>
        Panel.useSelectSelection({ key: created.key, selected: [tabB.key] }),
      );
      expect(result.current).toEqual([tabB.key]);
      const first = result.current;
      const countBefore = renderCount();

      await act(async () => {
        await ops.result.current.dispatchAsync({
          key: created.key,
          actions: [panel.setTabView({ key: tabA.key, view: { type: "docs" } })],
        });
      });

      expect(result.current).toBe(first);
      expect(renderCount()).toEqual(countBefore);
    });

    it("should recompute when tab membership changes", async () => {
      const [tabA, tabB] = [newTab(), newTab()];
      const { created, ops } = await createTabs(tabA, tabB);
      const { result } = await loadAndUse(created.key, () =>
        Panel.useSelectSelection({ key: created.key, selected: [tabB.key] }),
      );
      expect(result.current).toEqual([tabB.key]);

      await act(async () => {
        await ops.result.current.dispatchAsync({
          key: created.key,
          actions: [panel.removeTab({ key: tabB.key })],
        });
      });

      await waitFor(() => expect(result.current).toEqual([tabA.key]));
    });
  });

  describe("structural and granular selectors", () => {
    const createTab = async (tab: panel.Tab) => {
      const created = await createPanel();
      const ops = await loadAndUse(created.key, () => Panel.useDispatch());
      await act(async () => {
        await ops.result.current.dispatchAsync({
          key: created.key,
          actions: [panel.insertTab({ tab, targetLeaf: panel.ROOT_NODE_KEY })],
        });
      });
      return { created, ops };
    };

    it("useSelectTabType does not re-render when only the args change", async () => {
      const tab = newTab();
      const { created, ops } = await createTab(tab);

      const type = await loadAndCount(created.key, () =>
        Panel.useSelectTabType({ key: created.key, tabKey: tab.key }),
      );
      const typeCountBefore = type.renderCount();

      await act(async () => {
        await ops.result.current.dispatchAsync({
          key: created.key,
          actions: [
            panel.setTabView({
              key: tab.key,
              view: { type: "selector", args: { path: "/y" } },
            }),
          ],
        });
      });

      expect(type.result.current).toEqual("selector");
      expect(type.renderCount()).toEqual(typeCountBefore);
    });

    it("useSelectTabType resolves a resource tab to its ontology type", async () => {
      const tab = newTab();
      const { created, ops } = await createTab(tab);
      const resource = { type: "schematic", key: uuid.create() } as const;

      const type = await loadAndCount(created.key, () =>
        Panel.useSelectTabType({ key: created.key, tabKey: tab.key }),
      );
      expect(type.result.current).toEqual("selector");

      await act(async () => {
        await ops.result.current.dispatchAsync({
          key: created.key,
          actions: [panel.setTabResource({ key: tab.key, resource })],
        });
      });
      await waitFor(() => expect(type.result.current).toEqual("schematic"));
    });

    it("useSetCurrentTabView dispatches set_tab_view for the scoped tab", async () => {
      const tab = newTab();
      const { created } = await createTab(tab);

      const composed = ({ children }: PropsWithChildren): ReactElement => {
        const Synnax = wrapper;
        return (
          <Synnax>
            <Panel.Scope.Provider value={created.key}>
              <Panel.TabScope.Provider value={tab.key}>
                {children}
              </Panel.TabScope.Provider>
            </Panel.Scope.Provider>
          </Synnax>
        );
      };

      const { result } = renderHook(
        () => ({
          setView: Panel.useSetCurrentTabView(),
          args: Panel.useSelectTabArgs({}),
        }),
        { wrapper: composed },
      );

      await act(async () => {
        result.current.setView({ type: "docs", args: { path: "/set" } });
      });
      await waitFor(() => expect(result.current.args).toEqual({ path: "/set" }));
    });

    it("useSetCurrentTabResource swaps the scoped tab to the resource", async () => {
      const tab = newTab();
      const { created } = await createTab(tab);
      const resource = { type: "lineplot", key: uuid.create() } as const;

      const composed = ({ children }: PropsWithChildren): ReactElement => {
        const Synnax = wrapper;
        return (
          <Synnax>
            <Panel.Scope.Provider value={created.key}>
              <Panel.TabScope.Provider value={tab.key}>
                {children}
              </Panel.TabScope.Provider>
            </Panel.Scope.Provider>
          </Synnax>
        );
      };

      // useSelectTabResource throws on a view tab, so the resource is read through
      // the on-demand getter only after the variant has flipped, the same way a
      // resource renderer only mounts once its tab is a resource tab.
      const { result } = renderHook(
        () => ({
          setResource: Panel.useSetCurrentTabResource(),
          getResource: Panel.useGetTabResource(),
          variant: Panel.useSelectTabVariant({}),
        }),
        { wrapper: composed },
      );
      expect(result.current.variant).toEqual("view");

      await act(async () => {
        result.current.setResource(resource);
      });
      await waitFor(() => expect(result.current.variant).toEqual("resource"));
      expect(result.current.getResource()).toEqual(resource);
    });
  });

  describe("node selectors", () => {
    const firstLeaf = panel.childNodeKey(panel.ROOT_NODE_KEY, "first");
    const lastLeaf = panel.childNodeKey(panel.ROOT_NODE_KEY, "last");

    it("useSelectNodeVariant reports the root variant and updates on a split", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        variant: Panel.useSelectNodeVariant({
          key: created.key,
          nodeKey: panel.ROOT_NODE_KEY,
        }),
        dispatch: Panel.useDispatch(),
      }));
      expect(result.current.variant).toEqual("leaf");

      const [tabA, tabB] = [newTab(), newTab()];
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_NODE_KEY }),
            panel.splitTab({ key: tabA.key, direction: "x" }),
          ],
        });
      });
      await waitFor(() => expect(result.current.variant).toEqual("split"));
    });

    it("useSelectNodeVariant does not re-render while the variant is unchanged", async () => {
      const { created, ops } = await createSplitPanel();
      const { result, renderCount } = await loadAndCount(created.key, () =>
        Panel.useSelectNodeVariant({ key: created.key, nodeKey: panel.ROOT_NODE_KEY }),
      );
      expect(result.current).toEqual("split");
      const countBefore = renderCount();

      await act(async () => {
        await ops.result.current.dispatchAsync({
          key: created.key,
          actions: [panel.resizeSplit({ split: panel.ROOT_NODE_KEY, size: 0.3 })],
        });
      });

      expect(result.current).toEqual("split");
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectLeafNode returns the leaf node with its tab keys", async () => {
      const { created, tabA } = await createSplitPanel();
      const { result } = await loadAndUse(created.key, () =>
        Panel.useSelectLeafNode({ key: created.key, nodeKey: lastLeaf }),
      );
      expect(result.current).toEqual({ variant: "leaf", tabs: [tabA.key] });
    });

    it("useSelectLeafNode resolves the panel key from scope", async () => {
      const { created, tabA } = await createSplitPanel();
      const composed = ({ children }: PropsWithChildren): ReactElement => {
        const Synnax = wrapper;
        return (
          <Synnax>
            <Panel.Scope.Provider value={created.key}>{children}</Panel.Scope.Provider>
          </Synnax>
        );
      };
      const { result } = renderHook(
        () => Panel.useSelectLeafNode({ nodeKey: lastLeaf }),
        { wrapper: composed },
      );
      expect(result.current.tabs).toEqual([tabA.key]);
    });

    it("useSelectLeafNode stays stable across a tab content change", async () => {
      const { created, ops, tabA } = await createSplitPanel();
      const { result, renderCount } = await loadAndCount(created.key, () =>
        Panel.useSelectLeafNode({ key: created.key, nodeKey: lastLeaf }),
      );
      const first = result.current;
      const countBefore = renderCount();

      await act(async () => {
        await ops.result.current.dispatchAsync({
          key: created.key,
          actions: [
            panel.setTabView({
              key: tabA.key,
              view: { type: "docs", args: { path: "/x" } },
            }),
          ],
        });
      });

      expect(result.current).toBe(first);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectLeafNode throws when the target node is a split", async () => {
      const { created } = await createSplitPanel();
      expect(() =>
        renderHook(
          () =>
            Panel.useSelectLeafNode({ key: created.key, nodeKey: panel.ROOT_NODE_KEY }),
          { wrapper },
        ),
      ).toThrow("not a leaf");
    });

    it("useSelectSplitNode returns the split node's direction", async () => {
      const { created } = await createSplitPanel();
      const { result } = await loadAndUse(created.key, () =>
        Panel.useSelectSplitNode({ key: created.key, nodeKey: panel.ROOT_NODE_KEY }),
      );
      expect(result.current.variant).toEqual("split");
      expect(result.current.direction).toEqual("x");
    });

    it("useSelectSplitNode updates the size after a resize", async () => {
      const { created, ops } = await createSplitPanel();
      const { result } = await loadAndUse(created.key, () =>
        Panel.useSelectSplitNode({ key: created.key, nodeKey: panel.ROOT_NODE_KEY }),
      );
      await act(async () => {
        await ops.result.current.dispatchAsync({
          key: created.key,
          actions: [panel.resizeSplit({ split: panel.ROOT_NODE_KEY, size: 0.25 })],
        });
      });
      await waitFor(() => expect(result.current.size).toEqual(0.25));
    });

    it("useSelectSplitNode throws when the target node is a leaf", async () => {
      const { created } = await createSplitPanel();
      expect(() =>
        renderHook(
          () => Panel.useSelectSplitNode({ key: created.key, nodeKey: firstLeaf }),
          { wrapper },
        ),
      ).toThrow("not a split");
    });
  });

  describe("tree selectors", () => {
    it("useSelectTabKeys returns every tab key sorted", async () => {
      const { created, tabA, tabB } = await createSplitPanel();
      const { result } = await loadAndUse(created.key, () =>
        Panel.useSelectTabKeys({ key: created.key }),
      );
      expect(result.current).toEqual([tabA.key, tabB.key].sort());
    });

    it("useSelectTabKeys does not re-render on a resize", async () => {
      const { created, ops } = await createSplitPanel();
      const { renderCount } = await loadAndCount(created.key, () =>
        Panel.useSelectTabKeys({ key: created.key }),
      );
      const countBefore = renderCount();

      await act(async () => {
        await ops.result.current.dispatchAsync({
          key: created.key,
          actions: [panel.resizeSplit({ split: panel.ROOT_NODE_KEY, size: 0.3 })],
        });
      });

      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectTabKeys recomputes when tab membership changes", async () => {
      const { created, ops, tabA, tabB } = await createSplitPanel();
      const { result } = await loadAndUse(created.key, () =>
        Panel.useSelectTabKeys({ key: created.key }),
      );
      expect(result.current).toEqual([tabA.key, tabB.key].sort());

      await act(async () => {
        await ops.result.current.dispatchAsync({
          key: created.key,
          actions: [panel.removeTab({ key: tabA.key })],
        });
      });

      await waitFor(() => expect(result.current).toEqual([tabB.key]));
    });

    it("useSelectName returns the name and updates after a rename", async () => {
      const created = await client.panels.create({ name: "name-before" });
      const ops = await loadAndUse(created.key, () => Panel.useRename());
      const { result } = await loadAndUse(created.key, () =>
        Panel.useSelectName({ key: created.key }),
      );
      expect(result.current).toEqual("name-before");

      await act(async () => {
        await ops.result.current.updateAsync({ key: created.key, name: "name-after" });
      });
      await waitFor(() => expect(result.current).toEqual("name-after"));
    });

    it("useSelectTabLeaf returns the leaf holding the given tab", async () => {
      const { created, tabA } = await createSplitPanel();
      const { result } = await loadAndUse(created.key, () =>
        Panel.useSelectTabLeaf({ key: created.key, tabKey: tabA.key }),
      );
      expect(result.current.variant).toEqual("leaf");
      expect(result.current.tabs.map((t) => t.key)).toEqual([tabA.key]);
    });

    it("useSelectTabLeaf keeps a stable reference when another leaf changes", async () => {
      const { created, ops, tabA, tabB } = await createSplitPanel();
      const { result, renderCount } = await loadAndCount(created.key, () =>
        Panel.useSelectTabLeaf({ key: created.key, tabKey: tabA.key }),
      );
      const first = result.current;
      const countBefore = renderCount();

      await act(async () => {
        await ops.result.current.dispatchAsync({
          key: created.key,
          actions: [panel.setTabView({ key: tabB.key, view: { type: "docs" } })],
        });
      });

      expect(result.current).toBe(first);
      expect(renderCount()).toEqual(countBefore);
    });
  });

  describe("useSingleDispatch", () => {
    it("dispatches a single action for the given key", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: Panel.useRetrieve({ key: created.key }),
        dispatch: Panel.useSingleDispatch({ key: created.key }),
      }));
      const tab = newTab();
      await act(async () => {
        result.current.dispatch(
          panel.insertTab({ tab, targetLeaf: panel.ROOT_NODE_KEY }),
        );
      });
      await waitFor(() =>
        expect(leafTabKeys(result.current.retrieve.data?.root)).toEqual([tab.key]),
      );
    });

    it("dispatches an array of actions in one call", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: Panel.useRetrieve({ key: created.key }),
        dispatch: Panel.useSingleDispatch({ key: created.key }),
      }));
      const [tabA, tabB] = [newTab(), newTab()];
      await act(async () => {
        result.current.dispatch([
          panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY }),
          panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_NODE_KEY }),
        ]);
      });
      await waitFor(() =>
        expect(leafTabKeys(result.current.retrieve.data?.root)).toEqual([
          tabA.key,
          tabB.key,
        ]),
      );
    });

    it("resolves the panel key from the surrounding scope", async () => {
      const created = await createPanel();
      const composed = ({ children }: PropsWithChildren): ReactElement => {
        const Synnax = wrapper;
        return (
          <Synnax>
            <Panel.Scope.Provider value={created.key}>{children}</Panel.Scope.Provider>
          </Synnax>
        );
      };
      const { result } = renderHook(
        () => ({
          retrieve: Panel.useRetrieve({ key: created.key }),
          dispatch: Panel.useSingleDispatch(),
        }),
        { wrapper: composed },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));

      const tab = newTab();
      await act(async () => {
        result.current.dispatch(
          panel.insertTab({ tab, targetLeaf: panel.ROOT_NODE_KEY }),
        );
      });
      await waitFor(() =>
        expect(leafTabKeys(result.current.retrieve.data?.root)).toEqual([tab.key]),
      );
    });
  });

  describe("reactive sync", () => {
    it("should propagate rename through the channel listener to useRetrieve", async () => {
      const target = await client.panels.create({ name: "reactive-before" });
      const { result } = renderHook(() => Panel.useRetrieve({ key: target.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.name).toEqual("reactive-before");

      await writer.panels.rename(target.key, "reactive-after");

      await waitFor(() => expect(result.current.data?.name).toEqual("reactive-after"));
    });

    it("applies dispatches from other writers through the action channel", async () => {
      const created = await createPanel();
      const { result } = await loadAndUse(created.key, () =>
        Panel.useSelectRoot({ key: created.key }),
      );
      expect(result.current.variant).toEqual("leaf");

      const [tabA, tabB] = [newTab(), newTab()];
      await writer.panels.retrieve(created.key);
      await writer.panels.dispatch(created.key, [
        panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_NODE_KEY }),
        panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_NODE_KEY }),
        panel.splitTab({ key: tabB.key, direction: "x" }),
      ]);

      await waitFor(() => {
        const root = asSplit(result.current);
        expect(root).toBeDefined();
        expect(leafTabKeys(root?.first)).toEqual([tabA.key]);
        expect(leafTabKeys(root?.last)).toEqual([tabB.key]);
      });
    });

    it("should propagate deletes through the channel listener", async () => {
      const target = await client.panels.create({ name: "reactive-delete" });
      const { result } = renderHook(() => Panel.useList(), { wrapper });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.data).toContain(target.key));

      await writer.panels.delete(target.key);

      await waitFor(() => expect(result.current.data).not.toContain(target.key));
    });
  });
});
