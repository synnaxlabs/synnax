// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Menu } from "@synnaxlabs/pluto";

import { useConnectModal } from "@/feature/opc/device/useConnectModal";
import { Task } from "@/feature/opc/task";
import { Device as PlatformDevice } from "@/platform/device";
import { type Ontology } from "@/platform/ontology";

const TASK_CONTEXT_MENU_ITEM_CONFIGS: PlatformDevice.TaskContextMenuItemConfig[] = [
  { itemKey: "opc.readTask", label: "Create read task", type: Task.READ_TYPE },
  { itemKey: "opc.writeTask", label: "Create write task", type: Task.WRITE_TYPE },
];

export const ContextMenuItems = (props: Ontology.TreeContextMenuProps) => {
  const connect = useConnectModal();
  const onConfigure = (deviceKey: device.Key) => connect({ deviceKey });
  return (
    <>
      <PlatformDevice.EditConnectionMenuItem {...props} onConfigure={onConfigure} />
      <Menu.Divider />
      <PlatformDevice.TaskContextMenuItems
        {...props}
        onConfigure={onConfigure}
        taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
      />
    </>
  );
};
