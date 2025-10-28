// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { type Layout } from "@/layout";
import { Create, CREATE_LAYOUT_TYPE } from "@/status/Create";
import { Explorer, EXPLORER_LAYOUT, EXPLORER_LAYOUT_TYPE } from "@/status/Explorer";
import { type View } from "@/view";

export * from "@/status/Create";
export * from "@/status/Explorer";
export * from "@/status/palette";
export * from "@/status/useListenForChanges";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CREATE_LAYOUT_TYPE]: Create,
  [EXPLORER_LAYOUT_TYPE]: Explorer,
};

export const VIEW_REGISTRY: View.Registry = {
  status: { icon: Icon.Status, layout: EXPLORER_LAYOUT },
};
