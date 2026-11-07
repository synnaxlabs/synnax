// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, lineplot, panel, project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { Panel } from "@/feature/panel";
import { Project } from "@/feature/project";
import { expandTreeRow, findTreeRow, renderLiveTree } from "@/platform/tree/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const renderPanelTree = async (
  appearance: Pick<panel.New, "icon" | "color"> = {},
): Promise<panel.Panel> => {
  const { key } = await client.projects.create({ name: uniqueName("project") });
  const pan = await client.panels.create({
    ...appearance,
    name: uniqueName("panel"),
    project: key,
  });
  await renderLiveTree({
    client,
    root: project.resourceID(key),
    items: { ...Project.TREE_ITEMS, ...Panel.TREE_ITEMS },
  });
  return pan;
};

const findRowIcon = async (
  pan: panel.Panel,
  icon: string,
): Promise<SVGElement | null> =>
  (await findTreeRow(pan.name)).querySelector<SVGElement>(`.lyra-icon--${icon}`);

describe("panel tree item", () => {
  it("should draw the default panel icon for a panel with none", async () => {
    const pan = await renderPanelTree();
    expect(await findRowIcon(pan, "panel")).not.toBeNull();
  });

  it("should draw the icon of the panel in its color", async () => {
    const pan = await renderPanelTree({ icon: "valve", color: "#dc136c" });
    const icon = await findRowIcon(pan, "valve");
    expect(icon?.style.getPropertyValue("--pluto-symbol-color")).toEqual(
      "220, 19, 108",
    );
  });

  it("should redraw the icon when another client changes it", async () => {
    const pan = await renderPanelTree();
    await findRowIcon(pan, "panel");
    const writer = createTestClient();
    await writer.panels.retrieve(pan.key);
    await writer.panels.dispatch(pan.key, panel.setIcon({ icon: "valve" }));
    await waitFor(async () => expect(await findRowIcon(pan, "valve")).not.toBeNull());
  });

  it("should list the visualizations its tabs display", async () => {
    const { key } = await client.projects.create({ name: uniqueName("project") });
    // The plot sits in another project, so only the panel lists it.
    const other = await client.projects.create({ name: uniqueName("project") });
    const plot = await client.lineplots.create({
      name: uniqueName("plot"),
      project: other.key,
    });
    const pan = await client.panels.create({
      name: uniqueName("panel"),
      project: key,
      root: {
        variant: "leaf",
        tabs: [
          {
            variant: "resource",
            key: uuid.create(),
            resource: lineplot.resourceID(plot.key),
          },
          { variant: "view", key: uuid.create(), type: "docs" },
        ],
      },
    });
    await renderLiveTree({
      client,
      root: project.resourceID(key),
      items: { ...Project.TREE_ITEMS, ...Panel.TREE_ITEMS, ...LinePlot.TREE_ITEMS },
    });
    await findTreeRow(pan.name);
    expandTreeRow(pan.name);
    expect(await findTreeRow(plot.name)).toBeDefined();
  });

  it("should list the tab records that the tree's client has not read", async () => {
    const { key } = await client.projects.create({ name: uniqueName("project") });
    const other = await client.projects.create({ name: uniqueName("project") });
    const plot = await client.lineplots.create({
      name: uniqueName("plot"),
      project: other.key,
    });
    const pan = await client.panels.create({
      name: uniqueName("panel"),
      project: key,
      root: {
        variant: "leaf",
        tabs: [
          {
            variant: "resource",
            key: uuid.create(),
            resource: lineplot.resourceID(plot.key),
          },
        ],
      },
    });
    // The client starts after the create, so no change of the plot reaches it.
    const reader = createTestClient();
    // The panel row holds a live read of the panel, so the tab read finds it cached.
    const release = reader.panels.onChange(pan.key, () => {});
    await reader.panels.retrieve(pan.key);
    await renderLiveTree({
      client: reader,
      root: project.resourceID(key),
      items: { ...Project.TREE_ITEMS, ...Panel.TREE_ITEMS, ...LinePlot.TREE_ITEMS },
    });
    await findTreeRow(pan.name);
    expandTreeRow(pan.name);
    expect(await findTreeRow(plot.name)).toBeDefined();
    release();
  });

  it("should keep the listed tabs when a tab list fails", async () => {
    const { key } = await client.projects.create({ name: uniqueName("project") });
    const [shown, added] = await Promise.all(
      ["shown", "added"].map(
        async (name) =>
          await client.lineplots.create({ name: uniqueName(name), project: key }),
      ),
    );
    const tab = (plot: lineplot.LinePlot): panel.Tab => ({
      variant: "resource",
      key: uuid.create(),
      resource: lineplot.resourceID(plot.key),
    });
    const pan = await client.panels.create({
      name: uniqueName("panel"),
      project: key,
      root: { variant: "leaf", tabs: [tab(shown)] },
    });
    const viewer = createTestClient();
    await renderLiveTree({
      client: viewer,
      projectClient: client,
      root: panel.resourceID(pan.key),
      items: { ...Panel.TREE_ITEMS, ...LinePlot.TREE_ITEMS },
    });
    await findTreeRow(shown.name);
    const error = vi.spyOn(console, "error").mockImplementation(() => {});
    await viewer.panels.dispatch(pan.key, panel.insertTabs({ tabs: [tab(added)] }));
    // The new tab starts a list of the plots, and the close fails it.
    await viewer.close();
    await waitFor(() => expect(error).toHaveBeenCalledOnce());
    expect(DisconnectedError.matches(error.mock.calls[0][0])).toBe(true);
    expect(await findTreeRow(shown.name)).toBeDefined();
  });
});
