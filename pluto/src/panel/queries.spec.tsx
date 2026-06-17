// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, panel, project } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor, within } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { Errors } from "@/errors";
import { Panel } from "@/panel";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const newTab = (): panel.Tab => ({ key: uuid.create(), type: "selector", args: {} });

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
    // not followed by additional hooks — that shape trips a React 19
    // concurrent-replay warning.
    it("populates the store so downstream selectors resolve", async () => {
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

    it("should store the created panel in the flux store", async () => {
      const { result } = renderHook(() => Panel.useCreate(), { wrapper });
      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({ key, name: "stored-panel" });
      });

      const { result: root } = renderHook(() => Panel.useSelectRoot({ key }), {
        wrapper,
      });
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
          actions: [panel.insertTab({ tab, targetLeaf: panel.ROOT_PATH })],
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
          actions: [panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_PATH })],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.insertTab({
              tab: tabB,
              targetLeaf: panel.ROOT_PATH,
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
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.splitLeaf({ leaf: panel.ROOT_PATH, location: "right" })],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.resizeSplit({ split: panel.ROOT_PATH, size: 0.25 })],
        });
      });
      await waitFor(() =>
        expect(asSplit(result.current.retrieve.data?.root)?.size).toEqual(0.25),
      );

      const fresh = await client.panels.retrieve(created.key);
      expect(asSplit(fresh.root)?.size).toEqual(0.25);
    });

    it("moves a tab into the sibling leaf created by a split", async () => {
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
            panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_PATH }),
            panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_PATH }),
          ],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.splitLeaf({ leaf: panel.ROOT_PATH, location: "right" })],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.moveTab({
              key: tabA.key,
              targetLeaf: panel.childPath(panel.ROOT_PATH, "last"),
            }),
          ],
        });
      });
      await waitFor(() => {
        const root = asSplit(result.current.retrieve.data?.root);
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
          actions: [panel.insertTab({ tab, targetLeaf: panel.ROOT_PATH })],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.moveTab({
              key: tab.key,
              targetLeaf: panel.ROOT_PATH,
              location: "right",
            }),
          ],
        });
      });
      const root = result.current.retrieve.data?.root;
      expect(root?.variant).toEqual("leaf");
      expect(leafTabKeys(root)).toEqual([tab.key]);

      const fresh = await client.panels.retrieve(created.key);
      expect(fresh.root).toEqual(root);
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
            panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_PATH }),
            panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_PATH }),
          ],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.splitLeaf({ leaf: panel.ROOT_PATH, location: "right" })],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.moveTab({
              key: tabA.key,
              targetLeaf: panel.childPath(panel.ROOT_PATH, "last"),
            }),
          ],
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

    it("set_tab_type and set_tab_args replace a tab's type and args and persist them", async () => {
      const created = await createPanel();
      const tab = newTab();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: Panel.useRetrieve({ key: created.key }),
        dispatch: Panel.useDispatch(),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.insertTab({ tab, targetLeaf: panel.ROOT_PATH })],
        });
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.setTabType({ key: tab.key, type: "docs" }),
            panel.setTabArgs({ key: tab.key, args: { path: "/intro" } }),
          ],
        });
      });
      const updated = panel.findTab(result.current.retrieve.data!.root, tab.key);
      expect(updated?.type).toEqual("docs");
      expect(updated?.args).toEqual({ path: "/intro" });

      const fresh = await client.panels.retrieve(created.key);
      const freshTab = panel.findTab(fresh.root, tab.key);
      expect(freshTab?.type).toEqual("docs");
      expect(freshTab?.args).toEqual({ path: "/intro" });
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
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.splitLeaf({ leaf: panel.ROOT_PATH, location: "right" })],
        });
      });
      for (const size of [0.2, 0.3, 0.4])
        await act(async () => {
          await result.current.dispatch.dispatchAsync({
            key: created.key,
            actions: [panel.resizeSplit({ split: panel.ROOT_PATH, size })],
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
      const firstChild = panel.childPath(panel.ROOT_PATH, "first");
      const dispatchOne = async (action: panel.Action) =>
        await act(async () => {
          await result.current.dispatch.dispatchAsync({
            key: created.key,
            actions: [action],
          });
        });
      await dispatchOne(panel.splitLeaf({ leaf: panel.ROOT_PATH, location: "right" }));
      await dispatchOne(panel.splitLeaf({ leaf: firstChild, location: "right" }));
      await dispatchOne(panel.resizeSplit({ split: panel.ROOT_PATH, size: 0.3 }));
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
      const [tabA, tabB] = [newTab(), newTab()];
      const firstChild = panel.childPath(panel.ROOT_PATH, "first");
      const lastChild = panel.childPath(panel.ROOT_PATH, "last");
      const dispatchOne = async (action: panel.Action) =>
        await act(async () => {
          await result.current.dispatch.dispatchAsync({
            key: created.key,
            actions: [action],
          });
        });
      await dispatchOne(panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_PATH }));
      await dispatchOne(panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_PATH }));
      await dispatchOne(panel.splitLeaf({ leaf: panel.ROOT_PATH, location: "right" }));
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
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.splitLeaf({ leaf: panel.ROOT_PATH, location: "right" })],
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
          actions: [panel.resizeSplit({ split: panel.ROOT_PATH, size: 0.7 })],
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

      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.splitLeaf({ leaf: panel.ROOT_PATH, location: "right" })],
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

    it("useSelectTab returns the tab and updates after set_tab_type", async () => {
      const created = await createPanel();
      const tab = newTab();
      const ops = await loadAndUse(created.key, () => ({
        dispatch: Panel.useDispatch(),
      }));
      await act(async () => {
        await ops.result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [panel.insertTab({ tab, targetLeaf: panel.ROOT_PATH })],
        });
      });

      const { result } = renderHook(
        () => Panel.useSelectTab({ key: created.key, tabKey: tab.key }),
        { wrapper },
      );
      expect(result.current.key).toEqual(tab.key);
      expect(result.current.type).toEqual("selector");

      await act(async () => {
        await ops.result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            panel.setTabType({ key: tab.key, type: "docs" }),
            panel.setTabArgs({ key: tab.key, args: { path: "/x" } }),
          ],
        });
      });
      await waitFor(() => expect(result.current.type).toEqual("docs"));
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
            panel.insertTab({ tab: tabA, targetLeaf: panel.ROOT_PATH }),
            panel.insertTab({ tab: tabB, targetLeaf: panel.ROOT_PATH }),
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
          actions: [panel.setTabType({ key: tabB.key, type: "docs" })],
        });
      });

      expect(result.current).toBe(firstTab);
      expect(renderCount()).toEqual(countBefore);
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

      await client.panels.rename(target.key, "reactive-after");

      await waitFor(() => expect(result.current.data?.name).toEqual("reactive-after"));
    });

    it("applies dispatches from other writers through the action channel", async () => {
      const created = await createPanel();
      // Seed the root with a tab so the split's original side stays non-empty;
      // splitting an empty leaf and filling only the new sibling collapses the
      // empty pane back to a single leaf.
      const seed = newTab();
      await client.panels.dispatch(created.key, "", [
        panel.insertTab({ tab: seed, targetLeaf: panel.ROOT_PATH }),
      ]);
      const { result } = await loadAndUse(created.key, () =>
        Panel.useSelectRoot({ key: created.key }),
      );
      expect(leafTabKeys(result.current)).toEqual([seed.key]);

      const tab = newTab();
      await client.panels.dispatch(created.key, "", [
        panel.splitLeaf({ leaf: panel.ROOT_PATH, location: "right" }),
        panel.insertTab({
          tab,
          targetLeaf: panel.childPath(panel.ROOT_PATH, "last"),
        }),
      ]);

      await waitFor(() => {
        const root = asSplit(result.current);
        expect(root).toBeDefined();
        expect(leafTabKeys(root?.first)).toEqual([seed.key]);
        expect(leafTabKeys(root?.last)).toEqual([tab.key]);
      });
    });

    it("should propagate deletes through the channel listener", async () => {
      const target = await client.panels.create({ name: "reactive-delete" });
      const { result } = renderHook(() => Panel.useList(), { wrapper });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.data).toContain(target.key));

      await client.panels.delete(target.key);

      await waitFor(() => expect(result.current.data).not.toContain(target.key));
    });
  });
});
