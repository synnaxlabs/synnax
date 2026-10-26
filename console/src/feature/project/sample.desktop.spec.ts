// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type arc,
  lineplot,
  type panel,
  project,
  schematic,
  TimeStamp,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Project } from "@/feature/project";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";
import { type TestStore } from "@/testutil";

const client = createTestClient();

const CONFIRM = "Are you sure you want to reset the sample project?";

interface Sample {
  project: project.Key;
  panel: panel.Key;
  schematic: schematic.Key;
  linePlot: lineplot.Key;
  arcs: arc.Key[];
}

const resourceKeys = (node: panel.Node): string[] => {
  if (node.variant !== "leaf") throw new Error("expected a leaf");
  return node.tabs.flatMap((tab) =>
    tab.variant === "resource" ? [tab.resource.key] : [],
  );
};

const retrieveSample = async (key: project.Key): Promise<Sample> => {
  const panels = await client.panels.retrieve({ parent: project.ontologyID(key) });
  expect(panels).toHaveLength(1);
  const { key: panelKey, root } = panels[0];
  if (root.variant !== "split" || root.first.variant !== "split")
    throw new Error("expected a split panel");
  const [sch] = resourceKeys(root.first.first);
  const [plot] = resourceKeys(root.first.last);
  return {
    project: key,
    panel: panelKey,
    schematic: sch,
    linePlot: plot,
    arcs: resourceKeys(root.last),
  };
};

const retrieveSequence = async (sample: Sample): Promise<arc.Arc> => {
  const arcs = await client.arcs.retrieve(sample.arcs);
  const sequence = arcs.find((a) => a.name === "Press sequence");
  if (sequence == null) throw new Error("expected a Press sequence");
  return sequence;
};

describe("Sample project", () => {
  let created: Sample | undefined;

  const waitForSample = async (store: TestStore): Promise<Sample> => {
    await waitFor(() =>
      expect(Session.Project.selectOptionalSelected(store.getState())).toBeDefined(),
    );
    created = await retrieveSample(Session.Project.selectSelected(store.getState()));
    return created;
  };

  const openFromPalette = async (store?: TestStore) => {
    const palette = await renderPalette({ commands: Project.COMMANDS, client, store });
    // An open reports nothing when it finishes, but its last step takes the schematic
    // out of edit mode. Marking the schematic editable first makes that step a signal.
    const open = async () => {
      if (created != null)
        palette.store.dispatch(
          Session.Schematic.setEditable({ key: created.schematic, editable: true }),
        );
      await palette.openCommandPalette();
      await palette.selectCommand("Open sample project");
    };
    // A spec that opens again waits for the open to finish, so no open outlives it.
    const waitForOpened = async (sample: Sample) =>
      await waitFor(() =>
        expect(
          Session.Schematic.selectState({
            state: palette.store.getState(),
            key: sample.schematic,
          }).editable,
        ).toBe(false),
      );
    await open();
    const sample = await waitForSample(palette.store);
    return { store: palette.store, sample, open, waitForOpened };
  };

  afterEach(async () => {
    if (created == null) return;
    await client.arcs.delete(created.arcs);
    await client.panels.delete(created.panel);
    await client.schematics.delete(created.schematic);
    await client.lineplots.delete(created.linePlot);
    await client.projects.delete(created.project);
    created = undefined;
  });

  it("should open the schematic out of edit mode after the project loads", async () => {
    const { store, sample } = await openFromPalette(
      await Session.createStore({
        enablePrerender: false,
        openKV: () => new Session.Persist.MemoryKV(),
      }),
    );
    await waitFor(() =>
      expect(Session.Persist.selectSwapping(store.getState())).toBe(false),
    );
    expect(
      documentIn(store.getState()[Session.Schematic.SLICE_NAME], sample.schematic)
        ?.editable,
    ).toBe(false);
  });

  it("should open bound to the sample channels", async () => {
    const { sample } = await openFromPalette();
    expect((await client.projects.retrieve({ key: sample.project })).name).toBe(
      "Sample project",
    );
    const names = [
      "sample_time",
      "sample_tank_pt",
      "sample_press_vlv_cmd",
      "sample_press_vlv_state",
    ];
    const channels = await client.channels.retrieve(names);
    const [time, pt, cmd, state] = names.map(
      (name) => channels.find((ch) => ch.name === name)?.key,
    );
    const sch = await client.schematics.retrieve({ key: sample.schematic });
    expect(sch.configs.press_vlv).toMatchObject({
      variant: "valve",
      commandChannel: cmd,
      stateChannel: state,
    });
    expect(sch.configs.tank).toMatchObject({ variant: "tank", channel: pt });
    expect(sch.configs.tank_pt).toMatchObject({ variant: "value", channel: pt });
    const plot = await client.lineplots.retrieve({ key: sample.linePlot });
    expect(plot.channels.x1).toBe(time);
    expect(plot.channels.y1).toEqual([pt]);
    expect(sample.arcs).toHaveLength(2);
  });

  it("should pressurize after a Start at the schematic authority", async () => {
    await openFromPalette();
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: [
        "sample_start_cmd",
        "sample_abort_cmd",
        "sample_press_vlv_cmd",
        "sample_vent_vlv_cmd",
      ],
      authorities: 1,
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
  });

  it("should vent on an Abort while an operator holds the vent valve", async () => {
    await openFromPalette();
    // A control chip takes its valve at 255.
    const operator = await client.openWriter({
      start: TimeStamp.now(),
      channels: ["sample_start_cmd", "sample_abort_cmd", "sample_vent_vlv_cmd"],
      authorities: [1, 1, 255],
    });
    const streamer = await client.openStreamer([
      "sample_press_vlv_state",
      "sample_vent_vlv_state",
    ]);
    const waitForOpen = async (valve: string): Promise<void> => {
      let state = 0;
      while (state !== 1) state = Number((await streamer.read()).get(valve).at(-1));
    };
    try {
      await operator.write("sample_vent_vlv_cmd", 0);
      await operator.write("sample_start_cmd", 1);
      await waitForOpen("sample_press_vlv_state");
      await operator.write("sample_abort_cmd", 1);
      await waitForOpen("sample_vent_vlv_state");
    } finally {
      streamer.close();
      await operator.close();
    }
  });

  it("should reset the sample after the user confirms", async () => {
    const { sample, open, waitForOpened } = await openFromPalette();
    await client.projects.rename(sample.project, "My tank");
    const sequence = await retrieveSequence(sample);
    await client.arcs.create({
      ...sequence,
      text: { raw: sequence.text.raw.replace("duration=10s", "duration=20s") },
    });
    await open();
    expect(await screen.findByText(CONFIRM)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Reset" }));
    await waitFor(async () =>
      expect((await client.projects.retrieve({ key: sample.project })).name).toBe(
        "Sample project",
      ),
    );
    await waitFor(async () =>
      expect((await retrieveSequence(sample)).text.raw).toContain("duration=10s"),
    );
    await waitForOpened(sample);
    expect((await retrieveSample(sample.project)).arcs).toEqual(sample.arcs);
  });

  it("should keep the sample when the user cancels the reset", async () => {
    const { sample, open } = await openFromPalette();
    await client.projects.rename(sample.project, "My tank");
    await open();
    fireEvent.click(await screen.findByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByText(CONFIRM)).toBeNull());
    // A wrong reset renames the project before a second confirm can show.
    await open();
    expect(await screen.findByText(CONFIRM)).toBeTruthy();
    expect((await client.projects.retrieve({ key: sample.project })).name).toBe(
      "My tank",
    );
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByText(CONFIRM)).toBeNull());
  });

  it("should restart a running test on a reset", async () => {
    const { sample, open, waitForOpened } = await openFromPalette();
    const writer = await client.openWriter({
      start: TimeStamp.now(),
      channels: ["sample_start_cmd"],
      authorities: 1,
    });
    const streamer = await client.openStreamer(["sample_tank_pt"]);
    const readPressure = async (): Promise<number> =>
      Number((await streamer.read()).get("sample_tank_pt").at(-1));
    try {
      await writer.write("sample_start_cmd", 1);
      let pressure = 0;
      while (pressure < 50) pressure = await readPressure();
      await open();
      fireEvent.click(await screen.findByRole("button", { name: "Reset" }));
      // The simulator starts with an empty tank.
      while (pressure !== 0) pressure = await readPressure();
      await waitForOpened(sample);
    } finally {
      streamer.close();
      await writer.close();
    }
  });

  it("should relink the schematic and plot after a project delete", async () => {
    const { sample, open, waitForOpened } = await openFromPalette();
    await client.projects.delete(sample.project);
    await open();
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({
        ids: project.ontologyID(sample.project),
      });
      const ids = children.map(({ id }) => id);
      expect(ids).toContainEqual(schematic.ontologyID(sample.schematic));
      expect(ids).toContainEqual(lineplot.ontologyID(sample.linePlot));
    });
    await waitForOpened(sample);
  });
});
