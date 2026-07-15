// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Tree } from "@/platform/tree";

const TreeItem = Tree.createItem({ type: "node", icon: <Icon.Node /> });

export const TREE_ITEMS = { node: TreeItem } satisfies Tree.Items;
