// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Policy } from "@/feature/access/policy";
import { Role } from "@/feature/access/role";
import { type Tree } from "@/platform/tree";

export * from "@/feature/access/policy";
export * from "@/feature/access/role";

export const TREE_ITEMS: Tree.Items = { ...Policy.TREE_ITEMS, ...Role.TREE_ITEMS };
