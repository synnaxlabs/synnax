// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, panel, project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

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
});
