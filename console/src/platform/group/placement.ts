// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type group, resource } from "@synnaxlabs/client";

/** Returns where a new group goes when created under the node of a tree. */
export const placement = (
  node: resource.ID,
  root: resource.ID,
): Pick<group.CreateParams, "parent" | "scope"> =>
  resource.idsEqual(node, root) ? { scope: root.type } : { parent: node };
