// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { type Device as PDevice, Flux, Icon, Menu } from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";

import { useToggleEnabled } from "@/feature/ethercat/device/queries";
import { type SlaveDevice } from "@/feature/ethercat/device/types";
import { useConfigureModal } from "@/feature/ethercat/device/useConfigureModal";
import { Task } from "@/feature/ethercat/task";
import { Device } from "@/platform/device";
import { type Ontology } from "@/platform/ontology";

const TASK_CONTEXT_MENU_ITEM_CONFIGS: Device.TaskContextMenuItemConfig[] = [
  {
    itemKey: "ethercat.readTask",
    label: "Create read task",
    layout: Task.READ_LAYOUT,
  },
  {
    itemKey: "ethercat.writeTask",
    label: "Create write task",
    layout: Task.WRITE_LAYOUT,
  },
];

export const ContextMenuItems = (props: Ontology.TreeContextMenuProps) => {
  const keys = props.selection.ids.map((id) => id.key);
  const store = Flux.useStore<PDevice.FluxSubStore>();
  const { update: toggleEnabled } = useToggleEnabled();
  const configure = useConfigureModal();
  const onConfigure = (deviceKey: device.Key) => configure({ deviceKey });

  const { allDisabled, allEnabled } = useMemo(() => {
    const devices = store.devices.get(keys) as SlaveDevice[];
    const disabledCount = devices.filter((d) => !d.properties?.enabled).length;
    return {
      allDisabled: disabledCount === devices.length,
      allEnabled: disabledCount === 0,
    };
  }, [store, keys]);

  const handleDisable = useCallback(() => {
    toggleEnabled({ keys, enabled: false });
  }, [keys, toggleEnabled]);

  const handleEnable = useCallback(() => {
    toggleEnabled({ keys, enabled: true });
  }, [keys, toggleEnabled]);

  return (
    <>
      <Device.ConfigureMenuItem {...props} onConfigure={onConfigure} />
      <Device.ChangeIdentifierMenuItem {...props} icon="Logo.EtherCAT" />
      <Menu.Divider />
      <Device.TaskContextMenuItems
        {...props}
        onConfigure={onConfigure}
        taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
      />
      <Menu.Divider />
      {!allDisabled && (
        <Menu.Item itemKey="ethercat.disable" onClick={handleDisable}>
          <Icon.Disable />
          Disable
        </Menu.Item>
      )}
      {!allEnabled && (
        <Menu.Item itemKey="ethercat.enable" onClick={handleEnable}>
          <Icon.Enable />
          Enable
        </Menu.Item>
      )}
    </>
  );
};
