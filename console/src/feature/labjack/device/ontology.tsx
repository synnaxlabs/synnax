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

import { useConfigureModal } from "@/feature/labjack/device/useConfigureModal";
import { Task } from "@/feature/labjack/task";
import { Device as PlatformDevice } from "@/platform/device";
import { type Ontology } from "@/platform/ontology";

const TASK_CONTEXT_MENU_ITEM_CONFIGS: PlatformDevice.TaskContextMenuItemConfig[] = [
  {
    itemKey: "labjack.readTask",
    label: "Create read task",
    type: Task.READ_TYPE,
  },
  {
    itemKey: "labjack.writeTask",
    label: "Create write task",
    type: Task.WRITE_TYPE,
  },
];

export const ContextMenuItems = (props: Ontology.TreeContextMenuProps) => {
  const configure = useConfigureModal();
  const onConfigure = (deviceKey: device.Key) => configure({ deviceKey });
  return (
    <>
      <PlatformDevice.ConfigureMenuItem {...props} onConfigure={onConfigure} />
      <PlatformDevice.ChangeIdentifierMenuItem {...props} icon="Logo.LabJack" />
      <Menu.Divider />
      <PlatformDevice.TaskContextMenuItems
        {...props}
        onConfigure={onConfigure}
        taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
      />
    </>
  );
};
