// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { AssignRole, ASSIGN_ROLE_LAYOUT_TYPE } from "@/user/AssignRole";
import { Register, REGISTER_LAYOUT_TYPE } from "@/user/Register";

export * from "@/user/AssignRole";
export * from "@/user/Badge";
export * from "@/user/Register";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [ASSIGN_ROLE_LAYOUT_TYPE]: AssignRole,
  [REGISTER_LAYOUT_TYPE]: Register,
};
