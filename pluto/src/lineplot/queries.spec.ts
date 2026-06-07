// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createTestClient,
  lineplot as lineplotClient,
  NotFoundError,
} from "@synnaxlabs/client";
import { color, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { LinePlot } from "@/lineplot";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("lineplot queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useRetrieve", () => {
    it("should retrieve a line plot by key", async () => {
      const workspace = await client.workspaces.create({
        name: "test_workspace",
        layout: {},
      });
      const plot = await client.lineplots.create(workspace.key, {
        name: "retrieve_test",
      });

      const { result } = renderHook(() => LinePlot.useRetrieve({ key: plot.key }), {
        wrapper,
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data?.key).toEqual(plot.key);
      expect(result.current.data?.name).toEqual("retrieve_test");
    });

    it("should cache retrieved line plots", async () => {
      const workspace = await client.workspaces.create({
        name: "cache_workspace",
        layout: {},
      });
      const plot = await client.lineplots.create(workspace.key, {
        name: "cached_plot",
      });

      const { result: result1 } = renderHook(
        () => LinePlot.useRetrieve({ key: plot.key }),
        { wrapper },
      );
      await waitFor(() => expect(result1.current.variant).toEqual("success"));

      const { result: result2 } = renderHook(
        () => LinePlot.useRetrieve({ key: plot.key }),
        { wrapper },
      );
      await waitFor(() => expect(result2.current.variant).toEqual("success"));
      expect(result2.current.data).toEqual(result1.current.data);
    });
  });

  describe("useCreate", () => {
    it("should create a new line plot", async () => {
      const workspace = await client.workspaces.create({
        name: "create_workspace",
        layout: {},
      });

      const { result } = renderHook(() => LinePlot.useCreate(), { wrapper });

      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({
          key,
          workspace: workspace.key,
          name: "created_plot",
        });
      });

      expect(result.current.variant).toEqual("success");
      expect(result.current.data?.name).toEqual("created_plot");
      expect(result.current.data?.workspace).toEqual(workspace.key);

      const retrieved = await client.lineplots.retrieve({ key });
      expect(retrieved.name).toEqual("created_plot");
    });

    it("should store created line plot in flux store", async () => {
      const workspace = await client.workspaces.create({
        name: "store_workspace",
        layout: {},
      });

      const { result: createResult } = renderHook(() => LinePlot.useCreate(), {
        wrapper,
      });

      const key = uuid.create();
      await act(async () => {
        await createResult.current.updateAsync({
          key,
          workspace: workspace.key,
          name: "stored_plot",
        });
      });

      const { result: retrieveResult } = renderHook(
        () => LinePlot.useRetrieve({ key }),
        { wrapper },
      );
      await waitFor(() => expect(retrieveResult.current.variant).toEqual("success"));
      expect(retrieveResult.current.data?.name).toEqual("stored_plot");
    });
  });

  describe("useRename", () => {
    it("should rename a line plot", async () => {
      const workspace = await client.workspaces.create({
        name: "rename_workspace",
        layout: {},
      });
      const plot = await client.lineplots.create(workspace.key, {
        name: "original_name",
      });

      const { result } = renderHook(
        () => {
          const retrieve = LinePlot.useRetrieve({ key: plot.key });
          const rename = LinePlot.useRename();
          return { retrieve, rename };
        },
        { wrapper },
      );

      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      expect(result.current.retrieve.data?.name).toEqual("original_name");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: plot.key,
          name: "renamed_plot",
        });
      });

      const retrieved = await client.lineplots.retrieve({ key: plot.key });
      expect(retrieved.name).toEqual("renamed_plot");
    });

    it("should update cached plot after rename", async () => {
      const workspace = await client.workspaces.create({
        name: "rename_cache_workspace",
        layout: {},
      });
      const plot = await client.lineplots.create(workspace.key, {
        name: "cache_original",
      });

      const { result } = renderHook(
        () => ({
          retrieve: LinePlot.useRetrieve({ key: plot.key }),
          rename: LinePlot.useRename(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));

      await act(async () => {
        await result.current.rename.updateAsync({
          key: plot.key,
          name: "cache_renamed",
        });
      });

      await waitFor(() => {
        expect(result.current.retrieve.data?.name).toEqual("cache_renamed");
      });
    });
  });

  describe("useDelete", () => {
    it("should delete a single line plot", async () => {
      const workspace = await client.workspaces.create({
        name: "delete_workspace",
        layout: {},
      });
      const plot = await client.lineplots.create(workspace.key, {
        name: "delete_single",
      });

      const { result } = renderHook(() => LinePlot.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(plot.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(client.lineplots.retrieve({ key: plot.key })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should delete multiple line plots", async () => {
      const workspace = await client.workspaces.create({
        name: "delete_multi_workspace",
        layout: {},
      });
      const plot1 = await client.lineplots.create(workspace.key, {
        name: "delete_multi_1",
      });
      const plot2 = await client.lineplots.create(workspace.key, {
        name: "delete_multi_2",
      });

      const { result } = renderHook(() => LinePlot.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync([plot1.key, plot2.key]);
      });

      expect(result.current.variant).toEqual("success");

      await expect(client.lineplots.retrieve({ key: plot1.key })).rejects.toThrow(
        NotFoundError,
      );
      await expect(client.lineplots.retrieve({ key: plot2.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("useDispatch", () => {
    const seedPlot = async () => {
      const ws = await client.workspaces.create({
        name: `dispatch_ws_${uuid.create()}`,
        layout: {},
      });
      return await client.lineplots.create(ws.key, { name: "dispatch_test" });
    };

    const loadAndUse = async <T>(key: string, hook: () => T) => {
      const retrieve = renderHook(() => LinePlot.useRetrieve({ key }), { wrapper });
      await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));
      return renderHook(hook, { wrapper });
    };

    it("applies a rename dispatch and updates the cached plot", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        retrieve: LinePlot.useRetrieve({ key: seeded.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [lineplotClient.rename({ name: "after_dispatch" })],
        });
      });
      await waitFor(() =>
        expect(result.current.retrieve.data?.name).toEqual("after_dispatch"),
      );
    });

    it("restores prior axis bounds after undoing a setAxis dispatch", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        retrieve: LinePlot.useRetrieve({ key: seeded.key }),
        dispatch: LinePlot.useDispatch(),
        undo: LinePlot.useUndo({ key: seeded.key }),
      }));
      await waitFor(() => expect(result.current.retrieve.data?.axes.x1).toBeDefined());
      const original = result.current.retrieve.data!.axes.x1;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [
            lineplotClient.setAxisBounds({
              key: original.key,
              bounds: { lower: 0, upper: 100 },
              autoBounds: { lower: false, upper: false },
            }),
          ],
        });
      });
      await waitFor(() =>
        expect(result.current.retrieve.data?.axes.x1.bounds.upper).toEqual(100),
      );
      await act(async () => result.current.undo.undo());
      await waitFor(() =>
        expect(result.current.retrieve.data?.axes.x1.bounds.upper).toEqual(
          original.bounds.upper,
        ),
      );
    });

    it("re-applies an axis change after undo then redo", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        retrieve: LinePlot.useRetrieve({ key: seeded.key }),
        dispatch: LinePlot.useDispatch(),
        undo: LinePlot.useUndo({ key: seeded.key }),
        redo: LinePlot.useRedo({ key: seeded.key }),
      }));
      await waitFor(() => expect(result.current.retrieve.data?.axes.x1).toBeDefined());
      const original = result.current.retrieve.data!.axes.x1;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [
            lineplotClient.setAxisBounds({
              key: original.key,
              bounds: { lower: -5, upper: 25 },
              autoBounds: { lower: false, upper: false },
            }),
          ],
        });
      });
      await waitFor(() =>
        expect(result.current.retrieve.data?.axes.x1.bounds.upper).toEqual(25),
      );
      await act(async () => result.current.undo.undo());
      await waitFor(() =>
        expect(result.current.retrieve.data?.axes.x1.bounds.upper).toEqual(
          original.bounds.upper,
        ),
      );
      await act(async () => result.current.redo.redo());
      await waitFor(() =>
        expect(result.current.retrieve.data?.axes.x1.bounds.upper).toEqual(25),
      );
    });

    it("coalesces a stream of setAxis on the same axis into a single undo step", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        retrieve: LinePlot.useRetrieve({ key: seeded.key }),
        dispatch: LinePlot.useDispatch(),
        undo: LinePlot.useUndo({ key: seeded.key }),
      }));
      await waitFor(() => expect(result.current.retrieve.data?.axes.x1).toBeDefined());
      const original = result.current.retrieve.data!.axes.x1;
      for (const upper of [20, 30, 40])
        await act(async () => {
          await result.current.dispatch.dispatchAsync({
            key: seeded.key,
            actions: [
              lineplotClient.setAxisBounds({
                key: original.key,
                bounds: { lower: 0, upper },
                autoBounds: { lower: false, upper: false },
              }),
            ],
          });
        });
      await waitFor(() =>
        expect(result.current.retrieve.data?.axes.x1.bounds.upper).toEqual(40),
      );
      await act(async () => result.current.undo.undo());
      await waitFor(() =>
        expect(result.current.retrieve.data?.axes.x1.bounds.upper).toEqual(
          original.bounds.upper,
        ),
      );
    });

    it("does not coalesce setAxis across different axes", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        retrieve: LinePlot.useRetrieve({ key: seeded.key }),
        dispatch: LinePlot.useDispatch(),
        undo: LinePlot.useUndo({ key: seeded.key }),
      }));
      await waitFor(() => expect(result.current.retrieve.data?.axes.x1).toBeDefined());
      const x1Orig = result.current.retrieve.data!.axes.x1;
      const x2Orig = result.current.retrieve.data!.axes.x2;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [
            lineplotClient.setAxisBounds({
              key: x1Orig.key,
              bounds: { lower: 0, upper: 50 },
              autoBounds: { lower: false, upper: false },
            }),
          ],
        });
      });
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [
            lineplotClient.setAxisBounds({
              key: x2Orig.key,
              bounds: { lower: 0, upper: 70 },
              autoBounds: { lower: false, upper: false },
            }),
          ],
        });
      });
      await waitFor(() => {
        expect(result.current.retrieve.data?.axes.x1.bounds.upper).toEqual(50);
        expect(result.current.retrieve.data?.axes.x2.bounds.upper).toEqual(70);
      });
      // One undo reverts only the most recent gesture (x2).
      await act(async () => result.current.undo.undo());
      await waitFor(() => {
        expect(result.current.retrieve.data?.axes.x1.bounds.upper).toEqual(50);
        expect(result.current.retrieve.data?.axes.x2.bounds.upper).toEqual(
          x2Orig.bounds.upper,
        );
      });
    });

    it("keeps setLine creates out of the undo stack so Cmd+Z never silently no-ops", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        retrieve: LinePlot.useRetrieve({ key: seeded.key }),
        dispatch: LinePlot.useDispatch(),
        undo: LinePlot.useUndo({ key: seeded.key }),
      }));
      // Establish a baseline undoable entry that undo MUST be able to revert.
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [lineplotClient.setTitle({ title: { level: "h4", visible: true } })],
        });
      });
      await waitFor(() =>
        expect(result.current.retrieve.data?.title.visible).toBe(true),
      );
      // setLine create — should NOT push to the undo stack.
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [
            lineplotClient.setLine({
              line: {
                key: "line-new",
                label: "ghost",
                color: color.construct("#ff0000"),
                strokeWidth: 2,
                downsample: 1,
                downsampleMode: "decimate",
              },
            }),
          ],
        });
      });
      await waitFor(() =>
        expect(
          result.current.retrieve.data?.lines.find((l) => l.key === "line-new"),
        ).toBeDefined(),
      );
      // Undo must revert the title change, NOT silently no-op the line create.
      await act(async () => result.current.undo.undo());
      await waitFor(() =>
        expect(result.current.retrieve.data?.title.visible).toBe(false),
      );
    });
  });

  describe("selectors", () => {
    const seedPlot = async () => {
      const ws = await client.workspaces.create({
        name: `selector_ws_${uuid.create()}`,
        layout: {},
      });
      return await client.lineplots.create(ws.key, { name: "selector_test" });
    };

    const loadAndUse = async <T>(key: string, hook: () => T) => {
      const retrieve = renderHook(() => LinePlot.useRetrieve({ key }), { wrapper });
      await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));
      return renderHook(hook, { wrapper });
    };

    it("useSelectName returns the name and updates after a rename", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        name: LinePlot.useSelectName({ key: seeded.key }),
        rename: LinePlot.useRename(),
      }));
      expect(result.current.name).toEqual("selector_test");
      await act(async () => {
        await result.current.rename.updateAsync({
          key: seeded.key,
          name: "selector_renamed",
        });
      });
      await waitFor(() => expect(result.current.name).toEqual("selector_renamed"));
    });

    it("useSelectTitle returns the title and updates after a setTitle dispatch", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        title: LinePlot.useSelectTitle({ key: seeded.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.title.visible).toBe(false);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [lineplotClient.setTitle({ title: { level: "h3", visible: true } })],
        });
      });
      await waitFor(() => {
        expect(result.current.title.visible).toBe(true);
        expect(result.current.title.level).toEqual("h3");
      });
    });

    it("useSelectLegend updates after a setLegend dispatch", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        legend: LinePlot.useSelectLegend({ key: seeded.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      const original = result.current.legend;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [lineplotClient.setLegendVisible({ visible: !original.visible })],
        });
      });
      await waitFor(() =>
        expect(result.current.legend.visible).toEqual(!original.visible),
      );
    });

    it("useSelectChannels updates after addChannel", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        channels: LinePlot.useSelectChannels({ key: seeded.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.channels.y1).toEqual([]);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [lineplotClient.addChannel({ axisKey: "y1", channel: 42 })],
        });
      });
      await waitFor(() => expect(result.current.channels.y1).toEqual([42]));
    });

    it("useSelectRanges updates after addRange", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        ranges: LinePlot.useSelectRanges({ key: seeded.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.ranges.x1).toEqual([]);
      const rangeKey = uuid.create();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [lineplotClient.addRange({ axisKey: "x1", range: rangeKey })],
        });
      });
      await waitFor(() => expect(result.current.ranges.x1).toEqual([rangeKey]));
    });

    it("useSelectAxes / useSelectAxis return per-axis data and update on dispatch", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        axes: LinePlot.useSelectAxes({ key: seeded.key }),
        axis: LinePlot.useSelectAxis({ key: seeded.key, axisKey: "y1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.axis.key).toEqual("y1");
      expect(result.current.axes.y1.key).toEqual("y1");
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [
            lineplotClient.setAxisLabel({
              key: result.current.axis.key,
              label: "Pressure",
            }),
          ],
        });
      });
      await waitFor(() => {
        expect(result.current.axis.label).toEqual("Pressure");
        expect(result.current.axes.y1.label).toEqual("Pressure");
      });
    });

    it("useSelectLines / useSelectLine / useSelectLineKeys reflect setLine", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        lines: LinePlot.useSelectLines({ key: seeded.key }),
        keys: LinePlot.useSelectLineKeys({ key: seeded.key }),
        line: LinePlot.useSelectLine({ key: seeded.key, lineKey: "ln-1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.lines).toEqual([]);
      expect(result.current.line).toBeUndefined();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [
            lineplotClient.setLine({
              line: {
                key: "ln-1",
                color: color.construct("#00aaff"),
                strokeWidth: 2,
                downsample: 1,
                downsampleMode: "decimate",
              },
            }),
          ],
        });
      });
      await waitFor(() => {
        expect(result.current.lines).toHaveLength(1);
        expect(result.current.keys).toEqual(["ln-1"]);
        expect(result.current.line?.color).toEqual(color.construct("#00aaff"));
      });
    });

    it("useSelectRules / useSelectRule reflect setRule and removeRule", async () => {
      const seeded = await seedPlot();
      const { result } = await loadAndUse(seeded.key, () => ({
        rules: LinePlot.useSelectRules({ key: seeded.key }),
        rule: LinePlot.useSelectRule({ key: seeded.key, ruleKey: "rl-1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.rules).toEqual([]);
      expect(result.current.rule).toBeUndefined();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [
            lineplotClient.setRule({
              rule: {
                key: "rl-1",
                label: "max",
                color: color.construct("#ff0000"),
                axis: "y1",
                lineWidth: 1,
                lineDash: 0,
                units: "psi",
                position: 4.5,
              },
            }),
          ],
        });
      });
      await waitFor(() => {
        expect(result.current.rules).toHaveLength(1);
        expect(result.current.rule?.position).toEqual(4.5);
      });
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: seeded.key,
          actions: [lineplotClient.removeRule({ key: "rl-1" })],
        });
      });
      await waitFor(() => {
        expect(result.current.rules).toEqual([]);
        expect(result.current.rule).toBeUndefined();
      });
    });
  });

  describe("useRetrieveObservableName", () => {
    it("fires the callback with the initial name and with each rename", async () => {
      const ws = await client.workspaces.create({
        name: `obs_name_ws_${uuid.create()}`,
        layout: {},
      });
      const seeded = await client.lineplots.create(ws.key, {
        name: "obs_initial",
      });
      const seen: string[] = [];
      const { result } = renderHook(
        () => ({
          obs: LinePlot.useRetrieveObservableName({
            onChange: (name) => seen.push(name),
          }),
          rename: LinePlot.useRename(),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.obs.retrieveAsync({ key: seeded.key });
      });
      await waitFor(() => expect(seen).toContain("obs_initial"));
      await act(async () => {
        await result.current.rename.updateAsync({
          key: seeded.key,
          name: "obs_renamed",
        });
      });
      await waitFor(() => expect(seen).toContain("obs_renamed"));
    });
  });
});
