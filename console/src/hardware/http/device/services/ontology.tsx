// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/http/device";
import { type Ontology } from "@/ontology";

const TASK_CONTEXT_MENU_ITEM_CONFIGS: Common.DeviceServices.TaskContextMenuItemConfig[] =
  [];

export const ContextMenuItems = (props: Ontology.TreeContextMenuProps) => (
  <Common.DeviceServices.ContextMenuItemsWithConfigure
    {...props}
    itemKey="http.connect"
    configureLayout={Device.CONNECT_LAYOUT}
    taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
  />
);
