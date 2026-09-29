// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, project, TimeStamp } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { Session } from "@/session";
import { renderHookWithConsole } from "@/testutil";

const client = createTestClient();

const retrievePanel = async (key: project.Key) => {
  const panels = await client.panels.retrieve({ parent: project.ontologyID(key) });
  expect(panels).toHaveLength(1);
  return panels[0];
};

const arcKeys = async (key: project.Key): Promise<arc.Key[]> => {
  const { root } = await retrievePanel(key);
  if (root.variant !== "split" || root.last.variant !== "split")
    throw new Error("expected a split panel");
  const leaf = root.last.last;
  if (leaf.variant !== "leaf") throw new Error("expected a leaf");
  return leaf.tabs.flatMap((tab) =>
    tab.variant === "resource" && tab.resource.type === "arc" ? [tab.resource.key] : [],
  );
};

describe("Sample project", () => {
  let created: project.Key[] = [];

  const createSample = async (): Promise<Project.Sample> => {
    const sample = await Project.createSample(client);
    created.push(sample.project);
    return sample;
  };

  afterEach(async () => {
    for (const key of created) {
      await client.arcs.delete(await arcKeys(key));
      await client.projects.delete(key);
    }
    created = [];
  });

  describe("createSample", () => {
    it("binds the schematic and line plot to the sample channels", async () => {
      const sample = await createSample();
      const channels = await client.channels.retrieve([
        "sample_time",
        "sample_tank_pt",
        "sample_press_vlv_cmd",
        "sample_press_vlv_state",
      ]);
      const [time, pt, cmd, state] = [
        "sample_time",
        "sample_tank_pt",
        "sample_press_vlv_cmd",
        "sample_press_vlv_state",
      ].map((name) => channels.find((ch) => ch.name === name)?.key);
      const sch = await client.schematics.retrieve({ key: sample.schematic });
      expect(sch.configs.press_vlv).toMatchObject({
        variant: "valve",
        commandChannel: cmd,
        stateChannel: state,
      });
      expect(sch.configs.tank_pt).toMatchObject({ variant: "value", channel: pt });
      const { root } = await retrievePanel(sample.project);
      if (root.variant !== "split" || root.last.variant !== "split")
        throw new Error("expected a split panel");
      const plotLeaf = root.last.first;
      if (plotLeaf.variant !== "leaf") throw new Error("expected a leaf");
      const plotTab = plotLeaf.tabs[0];
      if (plotTab.variant !== "resource") throw new Error("expected a resource tab");
      const plot = await client.lineplots.retrieve({ key: plotTab.resource.key });
      expect(plot.channels.x1).toBe(time);
      expect(plot.channels.y1).toEqual([pt]);
    });

    it("reuses the sample channels and creates a new project", async () => {
      const first = await createSample();
      const second = await createSample();
      expect(second.project).not.toBe(first.project);
      const channels = await client.channels.retrieve({ searchTerm: "sample_tank_pt" });
      expect(channels.filter((ch) => ch.name === "sample_tank_pt")).toHaveLength(1);
    });

    it("pressurizes the tank when the start command fires", async () => {
      const sample = await createSample();
      expect(await arcKeys(sample.project)).toHaveLength(2);
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: ["sample_start_cmd"],
      });
      const streamer = await client.openStreamer(["sample_tank_pt"]);
      try {
        await writer.write("sample_start_cmd", 1);
        let pressure = 0;
        while (pressure < 50) {
          const frame = await streamer.read();
          pressure = Number(frame.get("sample_tank_pt").at(-1));
        }
        expect(pressure).toBeGreaterThanOrEqual(50);
      } finally {
        streamer.close();
        await writer.close();
      }
    }, 30_000);
  });

  describe("useOpenSample", () => {
    it("selects the new project and gives its schematic override authority", async () => {
      const { result, store } = await renderHookWithConsole(
        () => Project.useOpenSample(),
        { client },
      );
      await act(async () => result.current());
      await waitFor(
        () =>
          expect(
            Session.Project.selectOptionalSelected(store.getState()),
          ).toBeDefined(),
        { timeout: 10_000 },
      );
      const key = Session.Project.selectSelected(store.getState());
      created.push(key);
      const { root } = await retrievePanel(key);
      if (root.variant !== "split" || root.first.variant !== "leaf")
        throw new Error("expected a split panel");
      const tab = root.first.tabs[0];
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      const state = Session.Schematic.selectState({
        state: store.getState(),
        key: tab.resource.key,
      });
      expect(state.control.authority).toBe(Project.SAMPLE_SCHEMATIC_AUTHORITY);
      expect(state.editable).toBe(false);
    });
  });
});
