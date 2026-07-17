// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Icon, Label } from "@synnaxlabs/pluto";

import { Tree } from "@/platform/tree";

const useName: Tree.UseName = (id: ontology.ID) =>
  Label.useRetrieve({ key: id.key }).data?.name ?? "";

const TreeItem = Tree.createItem({ type: "label", icon: <Icon.Label />, useName });

export const TREE_ITEMS = { label: TreeItem } satisfies Tree.Items;
