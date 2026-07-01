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

import { Device as CommonDevice } from "@/service/device";
import { useConfigureModal } from "@/service/ni/device/useConfigureModal";
import { Task } from "@/service/ni/task";
import { type Ontology } from "@/service/ontology";

const TASK_CONTEXT_MENU_ITEM_CONFIGS: CommonDevice.TaskContextMenuItemConfig[] = [
    {
      itemKey: "ni.analogReadTask",
      label: "Create analog read task",
      layout: Task.ANALOG_READ_LAYOUT,
    },
    {
      itemKey: "ni.analogWriteTask",
      label: "Create analog write task",
      layout: Task.ANALOG_WRITE_LAYOUT,
    },
    {
      itemKey: "ni.counterReadTask",
      label: "Create counter read task",
      layout: Task.COUNTER_READ_LAYOUT,
    },
    {
      itemKey: "ni.digitalReadTask",
      label: "Create digital read task",
      layout: Task.DIGITAL_READ_LAYOUT,
    },
    {
      itemKey: "ni.digitalWriteTask",
      label: "Create digital write task",
      layout: Task.DIGITAL_WRITE_LAYOUT,
    },
  ];

export const ContextMenuItems = (props: Ontology.TreeContextMenuProps) => {
  const configure = useConfigureModal();
  const onConfigure = (deviceKey: device.Key) => configure({ deviceKey });
  return (
    <>
      <CommonDevice.ConfigureMenuItem {...props} onConfigure={onConfigure} />
      <CommonDevice.ChangeIdentifierMenuItem {...props} icon="Logo.NI" />
      <Menu.Divider />
      <CommonDevice.TaskContextMenuItems
        {...props}
        onConfigure={onConfigure}
        taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
      />
    </>
  );
};
