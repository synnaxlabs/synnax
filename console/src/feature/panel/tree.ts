// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, panel } from "@synnaxlabs/client";
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

const TreeItem = Tree.createItem({ type: "panel", icon });

export const TREE_ITEMS: Tree.Items = { panel: TreeItem };
