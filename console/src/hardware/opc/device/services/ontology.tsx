// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Menu } from "@synnaxlabs/pluto";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/opc/device";
import { Task } from "@/hardware/opc/task";
import { Layout } from "@/layout";
import { type Ontology } from "@/ontology";

const TASK_CONTEXT_MENU_ITEM_CONFIGS: Common.DeviceServices.TaskContextMenuItemConfig[] =
  [
    { itemKey: "opc.readTask", label: "Create Read Task", layout: Task.READ_LAYOUT },
    { itemKey: "opc.writeTask", label: "Create Write Task", layout: Task.WRITE_LAYOUT },
  ];

export const ContextMenuItems = (props: Ontology.TreeContextMenuProps) => {
  const placeLayout = Layout.usePlacer();
  const {
    selection: { resources },
  } = props;
  if (resources.length !== 1) return null;
  const handleEditConnection = () =>
    placeLayout({ ...Device.CONNECT_LAYOUT, key: resources[0].id.key });
  return (
    <Common.DeviceServices.ContextMenuItems
      {...props}
      configureLayout={Device.CONNECT_LAYOUT}
      taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
    >
      <>
        <Menu.Item
          itemKey="opc.connect"
          startIcon={<Icon.Connect />}
          onClick={handleEditConnection}
        >
          Edit Connection
        </Menu.Item>
        <Menu.Divider />
      </>
    </Common.DeviceServices.ContextMenuItems>
  );
};
