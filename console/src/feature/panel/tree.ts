// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ontology,
  panel,
  type resource,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { type Icon } from "@synnaxlabs/lyra/icon";

import { renderIcon } from "@/feature/panel/icon";
import { type Group } from "@/platform/group";
import { Tree } from "@/platform/tree";

const appearanceZ = panel.panelZ.pick({ icon: true, color: true });

// The panel's tree resource carries the panel, so the tree item needs no retrieve.
const icon = (resource: ontology.Resource): Icon.ReactElement =>
  renderIcon(appearanceZ.parse(resource.data ?? {}));

/** Lists the panels that sit in a group. */
export const GROUP_MEMBER: Group.Member = {
  type: "panel",
  children: (client, key) =>
    Tree.createSource(client.panels, { groups: [key] }, Tree.toResource("panel")),
};

type Visualization = "schematic" | "lineplot" | "log" | "table";

// The panel a node carries can still hold the tab of a deleted resource, so a missing
// key is not an error.
const SOURCES: Record<Visualization, (client: Client, keys: string[]) => Tree.Source> =
  {
    schematic: (client, keys) =>
      Tree.createSource(
        client.schematics,
        { keys, ignoreNotFoundError: true },
        Tree.toResource("schematic"),
      ),
    lineplot: (client, keys) =>
      Tree.createSource(
        client.lineplots,
        { keys, ignoreNotFoundError: true },
        Tree.toResource("lineplot"),
      ),
    log: (client, keys) =>
      Tree.createSource(
        client.logs,
        { keys, ignoreNotFoundError: true },
        Tree.toResource("log"),
      ),
    table: (client, keys) =>
      Tree.createSource(
        client.tables,
        { keys, ignoreNotFoundError: true },
        Tree.toResource("table"),
      ),
  };

const VISUALIZATIONS = Object.keys(SOURCES) as Visualization[];

const tabResources = (node: panel.Node): resource.ID[] =>
  node.variant === "leaf"
    ? node.tabs.flatMap((tab) => (tab.variant === "resource" ? [tab.resource] : []))
    : [...tabResources(node.first), ...tabResources(node.last)];

const rootZ = panel.panelZ.pick({ root: true });

// The tabs come from the panel the node carries when it expands, so a tab added while
// the node is open lists on the next expand.
const children: Tree.Children = ({ client, resource }) => {
  const ids = tabResources(rootZ.parse(resource?.data ?? {}).root);
  return VISUALIZATIONS.flatMap((type) => {
    const keys = ids.filter((id) => id.type === type).map(({ key }) => key);
    return keys.length === 0 ? [] : [SOURCES[type](client, keys)];
  });
};

const TreeItem = Tree.createItem({ type: "panel", icon, children });

export const TREE_ITEMS: Tree.Items = { panel: TreeItem };
