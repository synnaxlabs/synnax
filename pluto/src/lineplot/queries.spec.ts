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
  DataType,
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
      const project = await client.projects.create({
        name: "test_project",
        layout: {},
      });
      const plot = await client.lineplots.create(project.key, {
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
      const project = await client.projects.create({
        name: "cache_project",
        layout: {},
      });
      const plot = await client.lineplots.create(project.key, {
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
      const project = await client.projects.create({
        name: "create_project",
        layout: {},
      });

      const { result } = renderHook(() => LinePlot.useCreate(), { wrapper });

      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({
          key,
          project: project.key,
          name: "created_plot",
        });
      });

      expect(result.current.variant).toEqual("success");
      expect(result.current.data?.name).toEqual("created_plot");
      expect(result.current.data?.project).toEqual(project.key);

      const retrieved = await client.lineplots.retrieve({ key });
      expect(retrieved.name).toEqual("created_plot");
    });

    it("should store created line plot in flux store", async () => {
      const project = await client.projects.create({
        name: "store_project",
        layout: {},
      });

      const { result: createResult } = renderHook(() => LinePlot.useCreate(), {
        wrapper,
      });

      const key = uuid.create();
      await act(async () => {
        await createResult.current.updateAsync({
          key,
          project: project.key,
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
      const project = await client.projects.create({
        name: "rename_project",
        layout: {},
      });
      const plot = await client.lineplots.create(project.key, {
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

      await waitFor(() =>
        expect(result.current.retrieve.data?.name).toEqual("renamed_plot"),
      );
      const retrieved = await client.lineplots.retrieve({ key: plot.key });
      expect(retrieved.name).toEqual("renamed_plot");
    });

    it("should update cached plot after rename", async () => {
      const project = await client.projects.create({
        name: "rename_cache_project",
        layout: {},
      });
      const plot = await client.lineplots.create(project.key, {
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
      const project = await client.projects.create({
        name: "delete_project",
        layout: {},
      });
      const plot = await client.lineplots.create(project.key, {
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
      const project = await client.projects.create({
        name: "delete_multi_project",
        layout: {},
      });
      const plot1 = await client.lineplots.create(project.key, {
        name: "delete_multi_1",
      });
      const plot2 = await client.lineplots.create(project.key, {
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
    const createPlot = async () => {
      const proj = await client.projects.create({
        name: `dispatch_ws_${uuid.create()}`,
        layout: {},
      });
      return await client.lineplots.create(proj.key, { name: "dispatch_test" });
    };

    const loadAndUse = async <T>(key: string, hook: () => T) => {
      const retrieve = renderHook(() => LinePlot.useRetrieve({ key }), { wrapper });
      await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));
      return renderHook(hook, { wrapper });
    };

    it("applies a rename dispatch and updates the cached plot", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: LinePlot.useRetrieve({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.rename({ name: "after_dispatch" })],
        });
      });
      await waitFor(() =>
        expect(result.current.retrieve.data?.name).toEqual("after_dispatch"),
      );
    });

    it("restores prior axis bounds after undoing a setAxis dispatch", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: LinePlot.useRetrieve({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
        undo: LinePlot.useUndo({ key: created.key }),
      }));
      await waitFor(() => expect(result.current.retrieve.data?.axes.x1).toBeDefined());
      const original = result.current.retrieve.data!.axes.x1;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setAxisBounds({
              key: original.key,
              bounds: { lower: 0, upper: 100 },
              manualBounds: { lower: true, upper: true },
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
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: LinePlot.useRetrieve({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
        undo: LinePlot.useUndo({ key: created.key }),
        redo: LinePlot.useRedo({ key: created.key }),
      }));
      await waitFor(() => expect(result.current.retrieve.data?.axes.x1).toBeDefined());
      const original = result.current.retrieve.data!.axes.x1;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setAxisBounds({
              key: original.key,
              bounds: { lower: -5, upper: 25 },
              manualBounds: { lower: true, upper: true },
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
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: LinePlot.useRetrieve({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
        undo: LinePlot.useUndo({ key: created.key }),
      }));
      await waitFor(() => expect(result.current.retrieve.data?.axes.x1).toBeDefined());
      const original = result.current.retrieve.data!.axes.x1;
      for (const upper of [20, 30, 40])
        await act(async () => {
          await result.current.dispatch.dispatchAsync({
            key: created.key,
            actions: [
              lineplotClient.setAxisBounds({
                key: original.key,
                bounds: { lower: 0, upper },
                manualBounds: { lower: true, upper: true },
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
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: LinePlot.useRetrieve({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
        undo: LinePlot.useUndo({ key: created.key }),
      }));
      await waitFor(() => expect(result.current.retrieve.data?.axes.x1).toBeDefined());
      const x1Orig = result.current.retrieve.data!.axes.x1;
      const x2Orig = result.current.retrieve.data!.axes.x2;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setAxisBounds({
              key: x1Orig.key,
              bounds: { lower: 0, upper: 50 },
              manualBounds: { lower: true, upper: true },
            }),
          ],
        });
      });
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setAxisBounds({
              key: x2Orig.key,
              bounds: { lower: 0, upper: 70 },
              manualBounds: { lower: true, upper: true },
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
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        retrieve: LinePlot.useRetrieve({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
        undo: LinePlot.useUndo({ key: created.key }),
      }));
      // Establish a baseline undoable entry that undo MUST be able to revert.
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setTitle({ title: { level: "h4", visible: true } })],
        });
      });
      await waitFor(() =>
        expect(result.current.retrieve.data?.title.visible).toBe(true),
      );
      // setLine create — should NOT push to the undo stack.
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
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
    const createPlot = async () => {
      const proj = await client.projects.create({
        name: `selector_ws_${uuid.create()}`,
        layout: {},
      });
      return await client.lineplots.create(proj.key, { name: "selector_test" });
    };

    const loadAndUse = async <T>(key: string, hook: () => T) => {
      const retrieve = renderHook(() => LinePlot.useRetrieve({ key }), { wrapper });
      await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));
      return renderHook(hook, { wrapper });
    };

    it("useSelectName returns the name and updates after a rename", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        name: LinePlot.useSelectName({ key: created.key }),
        rename: LinePlot.useRename(),
      }));
      expect(result.current.name).toEqual("selector_test");
      await act(async () => {
        await result.current.rename.updateAsync({
          key: created.key,
          name: "selector_renamed",
        });
      });
      await waitFor(() => expect(result.current.name).toEqual("selector_renamed"));
    });

    it("useSelectTitle returns the title and updates after a setTitle dispatch", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        title: LinePlot.useSelectTitle({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.title.visible).toBe(false);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setTitle({ title: { level: "h3", visible: true } })],
        });
      });
      await waitFor(() => {
        expect(result.current.title.visible).toBe(true);
        expect(result.current.title.level).toEqual("h3");
      });
    });

    it("useSelectLegend updates after a setLegend dispatch", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        legend: LinePlot.useSelectLegend({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      const original = result.current.legend;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setLegendHidden({ hidden: !original.hidden })],
        });
      });
      await waitFor(() =>
        expect(result.current.legend.hidden).toEqual(!original.hidden),
      );
    });

    it("useSelectRanges updates after addRange", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        ranges: LinePlot.useSelectRanges({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.ranges.x1).toEqual([]);
      const rangeKey = uuid.create();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.addRange({ axisKey: "x1", range: rangeKey })],
        });
      });
      await waitFor(() => expect(result.current.ranges.x1).toEqual([rangeKey]));
    });

    it("useSelectYAxisChannels returns one axis's channels and updates after setChannels", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        channels: LinePlot.useSelectYAxisChannels({ key: created.key, axisKey: "y1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.channels).toEqual([]);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setChannels({ axisKey: "y1", channels: [1, 2] })],
        });
      });
      await waitFor(() => expect(result.current.channels).toEqual([1, 2]));
    });

    it("useSelectXAxisChannel returns the single x channel and updates after setXChannel", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        channel: LinePlot.useSelectXAxisChannel({ key: created.key, axisKey: "x1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.channel).toEqual(0);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setXChannel({ axisKey: "x1", channel: 99 })],
        });
      });
      await waitFor(() => expect(result.current.channel).toEqual(99));
    });

    it("useSelectXAxisRanges returns one axis's ranges and updates after setRanges", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        ranges: LinePlot.useSelectXAxisRanges({ key: created.key, axisKey: "x1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.ranges).toEqual([]);
      const rangeKey = uuid.create();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setRanges({ axisKey: "x1", ranges: [rangeKey] })],
        });
      });
      await waitFor(() => expect(result.current.ranges).toEqual([rangeKey]));
    });

    it("useSelectAxisKeys returns displayed axes and updates as channels appear", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        axisKeys: LinePlot.useSelectAxisKeys({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.axisKeys).toEqual(["x1", "y1"]);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setChannels({ axisKey: "y2", channels: [7] })],
        });
      });
      await waitFor(() => expect(result.current.axisKeys).toEqual(["x1", "y1", "y2"]));
    });

    it("useSelectAxes / useSelectAxis return per-axis data and update on dispatch", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        axes: LinePlot.useSelectAxes({ key: created.key }),
        axis: LinePlot.useSelectAxis({ key: created.key, axisKey: "y1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.axis.key).toEqual("y1");
      expect(result.current.axes.y1.key).toEqual("y1");
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
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
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        lines: LinePlot.useSelectLines({ key: created.key }),
        keys: LinePlot.useSelectLineKeys({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.lines).toEqual([]);
      expect(result.current.keys).toEqual([]);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
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
      });

      const { result: line } = renderHook(
        () => LinePlot.useSelectLine({ key: created.key, lineKey: "ln-1" }),
        { wrapper },
      );
      expect(line.current.color).toEqual(color.construct("#00aaff"));
    });

    it("useSelectRules / useSelectRule reflect setRule and removeRule", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        rules: LinePlot.useSelectRules({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.rules).toEqual([]);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
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
      await waitFor(() => expect(result.current.rules).toHaveLength(1));

      const { result: rule, unmount } = renderHook(
        () => LinePlot.useSelectRule({ key: created.key, ruleKey: "rl-1" }),
        { wrapper },
      );
      expect(rule.current.position).toEqual(4.5);
      unmount();

      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.removeRule({ key: "rl-1" })],
        });
      });
      await waitFor(() => expect(result.current.rules).toEqual([]));
    });

    it("useSelectRule resolves a palette color for a rule with no stored color", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        rules: LinePlot.useSelectRules({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setRule({
              rule: {
                key: "rl-1",
                label: "max",
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
      await waitFor(() => expect(result.current.rules).toHaveLength(1));

      const { result: rule } = renderHook(
        () => LinePlot.useSelectRule({ key: created.key, ruleKey: "rl-1" }),
        { wrapper },
      );
      expect(rule.current.color).toBeDefined();
    });

    it("useSelectLineCount reflects the number of lines", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        count: LinePlot.useSelectLineCount({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.count).toEqual(0);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setLine({
              line: {
                key: "ln-count",
                color: color.construct("#00aaff"),
                strokeWidth: 2,
                downsample: 1,
                downsampleMode: "decimate",
              },
            }),
          ],
        });
      });
      await waitFor(() => expect(result.current.count).toEqual(1));
    });

    it("useSelectXAxisKeys / useSelectYAxisKeys reflect the displayed axes", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        xKeys: LinePlot.useSelectXAxisKeys({ key: created.key }),
        yKeys: LinePlot.useSelectYAxisKeys({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.xKeys).toEqual(["x1"]);
      expect(result.current.yKeys).toEqual(["y1"]);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setXChannel({ axisKey: "x2", channel: 9 }),
            lineplotClient.addChannel({ axisKey: "y2", channel: 11 }),
          ],
        });
      });
      await waitFor(() => {
        expect(result.current.xKeys).toEqual(["x1", "x2"]);
        expect(result.current.yKeys).toEqual(["y1", "y2"]);
      });
    });

    it("useSelectXAxis returns a resolved axis config and updates on dispatch", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        xAxis: LinePlot.useSelectXAxis({ key: created.key, axisKey: "x1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.xAxis.key).toEqual("x1");
      expect(result.current.xAxis.bounds).toBeDefined();
      expect(result.current.xAxis.type).toEqual("time");
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setAxisLabel({ key: "x1", label: "Time" })],
        });
      });
      await waitFor(() => expect(result.current.xAxis.label).toEqual("Time"));
    });

    it("useSelectXAxis derives a linear tick type from a non-timestamp channel", async () => {
      const suffix = uuid.create().replace(/-/g, "");
      const index = await client.channels.create({
        name: `lp_idx_${suffix}`,
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data = await client.channels.create({
        name: `lp_data_${suffix}`,
        dataType: DataType.FLOAT32,
        index: index.key,
      });
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        xAxis: LinePlot.useSelectXAxis({ key: created.key, axisKey: "x1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.xAxis.type).toEqual("time");
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setXChannel({ axisKey: "x1", channel: data.key })],
        });
      });
      await waitFor(() => expect(result.current.xAxis.type).toEqual("linear"));
    });

    it("useSelectYAxis returns axis, channels, and lineKeys", async () => {
      const created = await createPlot();
      const lineKey = lineplotClient.lineKey({
        yAxis: "y1",
        xAxis: "x1",
        range: uuid.create(),
        xChannel: 3,
        yChannel: 5,
      });
      const { result } = await loadAndUse(created.key, () => ({
        yAxis: LinePlot.useSelectYAxis({ key: created.key, axisKey: "y1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.yAxis.axis.key).toEqual("y1");
      expect(result.current.yAxis.channels).toEqual([]);
      expect(result.current.yAxis.lineKeys).toEqual([]);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.addChannel({ axisKey: "y1", channel: 5 }),
            lineplotClient.setLine({
              line: {
                key: lineKey,
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
        expect(result.current.yAxis.channels).toEqual([5]);
        expect(result.current.yAxis.lineKeys).toEqual([lineKey]);
      });
    });

    it("useSelectAxisRuleKeys returns the keys of rules on the given axis", async () => {
      const created = await createPlot();
      const { result } = await loadAndUse(created.key, () => ({
        y1Keys: LinePlot.useSelectAxisRuleKeys({ key: created.key, axisKey: "y1" }),
        y2Keys: LinePlot.useSelectAxisRuleKeys({ key: created.key, axisKey: "y2" }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.y1Keys).toEqual([]);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setRule({
              rule: {
                key: "rule-y1",
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
      await waitFor(() => expect(result.current.y1Keys).toEqual(["rule-y1"]));
      expect(result.current.y2Keys).toEqual([]);
    });
  });

  describe("selector stability", () => {
    const createPlot = async () => {
      const proj = await client.projects.create({
        name: `stability_ws_${uuid.create()}`,
        layout: {},
      });
      return await client.lineplots.create(proj.key, { name: "stability_test" });
    };

    const loadAndCount = async <T>(key: string, hook: () => T) => {
      const retrieve = renderHook(() => LinePlot.useRetrieve({ key }), { wrapper });
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

    it("useSelectTitle keeps a stable reference across an unrelated edit", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        title: LinePlot.useSelectTitle({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      const firstTitle = result.current.title;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.addChannel({ axisKey: "y1", channel: 99 })],
        });
      });
      expect(result.current.title).toBe(firstTitle);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectYAxisKeys stays stable when a channel is added to an already-displayed axis", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        yKeys: LinePlot.useSelectYAxisKeys({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.yKeys).toEqual(["y1"]);
      const firstKeys = result.current.yKeys;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.addChannel({ axisKey: "y1", channel: 42 })],
        });
      });
      expect(result.current.yKeys).toBe(firstKeys);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectLineKeys stays stable when a line's style changes", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        keys: LinePlot.useSelectLineKeys({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setLine({
              line: {
                key: "ln-stable",
                color: color.construct("#00aaff"),
                strokeWidth: 2,
                downsample: 1,
                downsampleMode: "decimate",
              },
            }),
          ],
        });
      });
      await waitFor(() => expect(result.current.keys).toEqual(["ln-stable"]));
      const firstKeys = result.current.keys;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setLineColor({
              key: "ln-stable",
              color: color.construct("#ff0000"),
            }),
          ],
        });
      });
      expect(result.current.keys).toBe(firstKeys);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectAxisRuleKeys stays stable when a rule's position changes", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        keys: LinePlot.useSelectAxisRuleKeys({ key: created.key, axisKey: "y1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setRule({
              rule: {
                key: "rule-stable",
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
      await waitFor(() => expect(result.current.keys).toEqual(["rule-stable"]));
      const firstKeys = result.current.keys;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setRulePosition({ key: "rule-stable", position: 9.9 }),
          ],
        });
      });
      expect(result.current.keys).toBe(firstKeys);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectName stays stable across an unrelated edit", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        name: LinePlot.useSelectName({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      const firstName = result.current.name;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setTitle({ title: { level: "h2", visible: true } })],
        });
      });
      expect(result.current.name).toBe(firstName);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectLegend keeps a stable reference across an unrelated edit", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        legend: LinePlot.useSelectLegend({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      const firstLegend = result.current.legend;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.addChannel({ axisKey: "y1", channel: 7 })],
        });
      });
      expect(result.current.legend).toBe(firstLegend);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectRanges keeps a stable reference across an unrelated edit", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        ranges: LinePlot.useSelectRanges({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      const firstRanges = result.current.ranges;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setTitle({ title: { level: "h2", visible: true } })],
        });
      });
      expect(result.current.ranges).toBe(firstRanges);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectAxes keeps a stable reference across an unrelated edit", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        axes: LinePlot.useSelectAxes({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      const firstAxes = result.current.axes;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setTitle({ title: { level: "h2", visible: true } })],
        });
      });
      expect(result.current.axes).toBe(firstAxes);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectAxis keeps a stable reference when a different axis changes", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        axis: LinePlot.useSelectAxis({ key: created.key, axisKey: "y1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      const firstAxis = result.current.axis;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setAxisLabel({ key: "x1", label: "Time" })],
        });
      });
      expect(result.current.axis).toBe(firstAxis);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectXAxisKeys stays stable when a channel is added to an already-displayed axis", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        xKeys: LinePlot.useSelectXAxisKeys({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.xKeys).toEqual(["x1"]);
      const firstKeys = result.current.xKeys;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.addChannel({ axisKey: "y1", channel: 42 })],
        });
      });
      expect(result.current.xKeys).toBe(firstKeys);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectXAxis keeps a stable reference across an unrelated edit", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        xAxis: LinePlot.useSelectXAxis({ key: created.key, axisKey: "x1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      expect(result.current.xAxis.type).toEqual("time");
      const firstXAxis = result.current.xAxis;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.addChannel({ axisKey: "y1", channel: 42 })],
        });
      });
      expect(result.current.xAxis).toBe(firstXAxis);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectYAxis keeps a stable reference across an unrelated edit", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        yAxis: LinePlot.useSelectYAxis({ key: created.key, axisKey: "y1" }),
        dispatch: LinePlot.useDispatch(),
      }));
      const firstYAxis = result.current.yAxis;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setTitle({ title: { level: "h2", visible: true } })],
        });
      });
      expect(result.current.yAxis).toBe(firstYAxis);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectLineCount stays stable across an edit that does not change the count", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        count: LinePlot.useSelectLineCount({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setLine({
              line: {
                key: "ln-count-stable",
                color: color.construct("#00aaff"),
                strokeWidth: 2,
                downsample: 1,
                downsampleMode: "decimate",
              },
            }),
          ],
        });
      });
      await waitFor(() => expect(result.current.count).toEqual(1));
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setLineColor({
              key: "ln-count-stable",
              color: color.construct("#ff0000"),
            }),
          ],
        });
      });
      expect(result.current.count).toEqual(1);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectLines keeps a stable reference across an unrelated edit", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        lines: LinePlot.useSelectLines({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setLine({
              line: {
                key: "ln-lines-stable",
                color: color.construct("#00aaff"),
                strokeWidth: 2,
                downsample: 1,
                downsampleMode: "decimate",
              },
            }),
          ],
        });
      });
      await waitFor(() => expect(result.current.lines).toHaveLength(1));
      const firstLines = result.current.lines;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setTitle({ title: { level: "h2", visible: true } })],
        });
      });
      expect(result.current.lines).toBe(firstLines);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectRules keeps a stable reference across an unrelated edit", async () => {
      const created = await createPlot();
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        rules: LinePlot.useSelectRules({ key: created.key }),
        dispatch: LinePlot.useDispatch(),
      }));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setRule({
              rule: {
                key: "rl-rules-stable",
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
      await waitFor(() => expect(result.current.rules).toHaveLength(1));
      const firstRules = result.current.rules;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setTitle({ title: { level: "h2", visible: true } })],
        });
      });
      expect(result.current.rules).toBe(firstRules);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectLine keeps a stable reference when a different line changes", async () => {
      const suffix = uuid.create().replace(/-/g, "");
      const index = await client.channels.create({
        name: `lnstab_idx_${suffix}`,
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const chA = await client.channels.create({
        name: `lnstab_a_${suffix}`,
        dataType: DataType.FLOAT32,
        index: index.key,
      });
      const chB = await client.channels.create({
        name: `lnstab_b_${suffix}`,
        dataType: DataType.FLOAT32,
        index: index.key,
      });
      const range = uuid.create();
      const lineA = lineplotClient.lineKey({
        yAxis: "y1",
        xAxis: "x1",
        range,
        xChannel: index.key,
        yChannel: chA.key,
      });
      const lineB = lineplotClient.lineKey({
        yAxis: "y1",
        xAxis: "x1",
        range,
        xChannel: index.key,
        yChannel: chB.key,
      });
      const created = await createPlot();
      const setup = renderHook(
        () => ({
          retrieve: LinePlot.useRetrieve({ key: created.key }),
          dispatch: LinePlot.useDispatch(),
        }),
        { wrapper },
      );
      await waitFor(() =>
        expect(setup.result.current.retrieve.variant).toEqual("success"),
      );
      await act(async () => {
        await setup.result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setLine({
              line: {
                key: lineA,
                color: color.construct("#00aaff"),
                strokeWidth: 2,
                downsample: 1,
                downsampleMode: "decimate",
              },
            }),
            lineplotClient.setLine({
              line: {
                key: lineB,
                color: color.construct("#ff8800"),
                strokeWidth: 2,
                downsample: 1,
                downsampleMode: "decimate",
              },
            }),
          ],
        });
      });
      let renderCount = 0;
      const { result } = renderHook(
        () => {
          renderCount++;
          return {
            line: LinePlot.useSelectLine({ key: created.key, lineKey: lineA }),
            dispatch: LinePlot.useDispatch(),
          };
        },
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current.line.label).toEqual(`lnstab_a_${suffix}`),
      );
      const firstLine = result.current.line;
      const countBefore = renderCount;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setLineColor({
              key: lineB,
              color: color.construct("#00ff00"),
            }),
          ],
        });
      });
      expect(result.current.line).toBe(firstLine);
      expect(renderCount).toEqual(countBefore);
    });

    it("useSelectRule keeps a stable reference when a different rule changes", async () => {
      const created = await createPlot();
      const setup = renderHook(
        () => ({
          retrieve: LinePlot.useRetrieve({ key: created.key }),
          dispatch: LinePlot.useDispatch(),
        }),
        { wrapper },
      );
      await waitFor(() =>
        expect(setup.result.current.retrieve.variant).toEqual("success"),
      );
      await act(async () => {
        await setup.result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            lineplotClient.setRule({
              rule: {
                key: "rl-a",
                label: "a",
                color: color.construct("#ff0000"),
                axis: "y1",
                lineWidth: 1,
                lineDash: 0,
                units: "psi",
                position: 1,
              },
            }),
            lineplotClient.setRule({
              rule: {
                key: "rl-b",
                label: "b",
                color: color.construct("#00ff00"),
                axis: "y1",
                lineWidth: 1,
                lineDash: 0,
                units: "psi",
                position: 2,
              },
            }),
          ],
        });
      });
      let renderCount = 0;
      const { result } = renderHook(
        () => {
          renderCount++;
          return {
            rule: LinePlot.useSelectRule({ key: created.key, ruleKey: "rl-a" }),
            dispatch: LinePlot.useDispatch(),
          };
        },
        { wrapper },
      );
      await waitFor(() => expect(result.current.rule?.position).toEqual(1));
      const firstRule = result.current.rule;
      const countBefore = renderCount;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [lineplotClient.setRulePosition({ key: "rl-b", position: 9 })],
        });
      });
      expect(result.current.rule).toBe(firstRule);
      expect(renderCount).toEqual(countBefore);
    });
  });

  describe("useRetrieveObservableName", () => {
    it("fires the callback with the initial name and with each rename", async () => {
      const proj = await client.projects.create({
        name: `obs_name_ws_${uuid.create()}`,
        layout: {},
      });
      const created = await client.lineplots.create(proj.key, {
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
        await result.current.obs.retrieveAsync({ key: created.key });
      });
      await waitFor(() => expect(seen).toContain("obs_initial"));
      await act(async () => {
        await result.current.rename.updateAsync({
          key: created.key,
          name: "obs_renamed",
        });
      });
      await waitFor(() => expect(seen).toContain("obs_renamed"));
    });
  });
});
