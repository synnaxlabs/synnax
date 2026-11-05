// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";

import { Tree } from "@/platform/tree";

const children: Tree.Children = ({ client, id }) => {
  const q = { hasLabels: [id.key] };
  return [
    Tree.createSource(client.ranges, q, (range: ranger.Range) =>
      Tree.toResource("range")(range.payload),
    ),
    Tree.createSource(client.statuses, q, Tree.toResource("status")),
  ];
};

const TreeItem = Tree.createItem({ type: "label", icon: <Icon.Label />, children });

export const TREE_ITEMS = { label: TreeItem } satisfies Tree.Items;
