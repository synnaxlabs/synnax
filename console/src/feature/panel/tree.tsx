// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, query } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Panel } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Appearance } from "@/platform/appearance";
import { type Group } from "@/platform/group";
import { Tree } from "@/platform/tree";

interface ItemIconProps {
  panelKey: panel.Key;
}

const ItemIcon = ({ panelKey }: ItemIconProps): ReactElement => {
  const { data } = Panel.useResultAppearance({ key: panelKey });
  return Appearance.renderIcon({ ...data, fallback: Icon.Panel });
};

/** Lists the panels that sit in a group. */
export const GROUP_MEMBER: Group.Member = {
  type: "panel",
  children: (client, key) =>
    Tree.createSource(client.panels, { groups: [key] }, Tree.toResource("panel")),
};

const FACTS = ["Panel"];

const haulPreview: Tree.HaulPreview = ({ resource, client, count }) => {
  const cached = client.panels.watch(resource.id.key).getCached();
  const appearance = query.isLive(cached) ? cached : {};
  return (
    <Haul.Preview
      icon={Appearance.renderIcon({ ...appearance, fallback: Icon.Panel })}
      name={resource.name}
      facts={FACTS}
      count={count}
    />
  );
};

const TreeItem = Tree.createItem({
  icon: ({ id }) => <ItemIcon panelKey={id.key} />,
  haulPreview,
});

export const TREE_ITEMS: Tree.Items = { panel: TreeItem };
