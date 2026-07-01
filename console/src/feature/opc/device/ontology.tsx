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

import { Device as CommonDevice } from "@/feature/device";
import { type Ontology } from "@/platform/ontology";
import { useConnectModal } from "@/feature/opc/device/useConnectModal";
import { Task } from "@/feature/opc/task";

const TASK_CONTEXT_MENU_ITEM_CONFIGS: CommonDevice.TaskContextMenuItemConfig[] = [
  { itemKey: "opc.readTask", label: "Create read task", layout: Task.READ_LAYOUT },
  { itemKey: "opc.writeTask", label: "Create write task", layout: Task.WRITE_LAYOUT },
];

export const ContextMenuItems = (props: Ontology.TreeContextMenuProps) => {
  const connect = useConnectModal();
  const onConfigure = (deviceKey: device.Key) => connect({ deviceKey });
  return (
    <>
      <CommonDevice.EditConnectionMenuItem {...props} onConfigure={onConfigure} />
      <Menu.Divider />
      <CommonDevice.TaskContextMenuItems
        {...props}
        onConfigure={onConfigure}
        taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
      />
    </>
  );
};
