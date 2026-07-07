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

import { useConfigureModal } from "@/feature/ni/device/useConfigureModal";
import { Task } from "@/feature/ni/task";
import { Device as PlatformDevice } from "@/platform/device";
import { type Tree } from "@/platform/tree";

const TASK_CONTEXT_MENU_ITEM_CONFIGS: PlatformDevice.TaskContextMenuItemConfig[] = [
  {
    itemKey: "ni.analogReadTask",
    label: "Create analog read task",
    type: Task.ANALOG_READ_TYPE,
  },
  {
    itemKey: "ni.analogWriteTask",
    label: "Create analog write task",
    type: Task.ANALOG_WRITE_TYPE,
  },
  {
    itemKey: "ni.counterReadTask",
    label: "Create counter read task",
    type: Task.COUNTER_READ_TYPE,
  },
  {
    itemKey: "ni.digitalReadTask",
    label: "Create digital read task",
    type: Task.DIGITAL_READ_TYPE,
  },
  {
    itemKey: "ni.digitalWriteTask",
    label: "Create digital write task",
    type: Task.DIGITAL_WRITE_TYPE,
  },
];

export const ContextMenuItems = (props: Tree.ContextMenuProps) => {
  const configure = useConfigureModal();
  const onConfigure = (deviceKey: device.Key) => configure({ deviceKey });
  return (
    <>
      <PlatformDevice.ConfigureMenuItem {...props} onConfigure={onConfigure} />
      <PlatformDevice.ChangeIdentifierMenuItem {...props} icon="Logo.NI" />
      <Menu.Divider />
      <PlatformDevice.TaskContextMenuItems
        {...props}
        onConfigure={onConfigure}
        taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
      />
    </>
  );
};
