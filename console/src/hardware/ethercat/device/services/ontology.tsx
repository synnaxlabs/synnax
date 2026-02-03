// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { type Device as PlutoDevice, Flux, Icon, Menu } from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ethercat/device";
import { useTogglePassive } from "@/hardware/ethercat/device/queries";
import { type SlaveProperties } from "@/hardware/ethercat/device/types";
import { Task } from "@/hardware/ethercat/task";
import { type Ontology } from "@/ontology";

const TASK_CONTEXT_MENU_ITEM_CONFIGS: Common.DeviceServices.TaskContextMenuItemConfig[] =
  [
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
  const store = Flux.useStore<PlutoDevice.FluxSubStore>();
  const { update: togglePassive } = useTogglePassive();

  const { allPassive, allActive } = useMemo(() => {
    const devices = store.devices.get(keys) as device.Device<SlaveProperties>[];
    const passiveCount = devices.filter((d) => d.properties?.passive).length;
    return {
      allPassive: passiveCount === devices.length,
      allActive: passiveCount === 0,
    };
  }, [store, keys]);

  const handleSetPassive = useCallback(() => {
    togglePassive({ keys, passive: true });
  }, [keys, togglePassive]);

  const handleSetActive = useCallback(() => {
    togglePassive({ keys, passive: false });
  }, [keys, togglePassive]);

  return (
    <>
      <Common.DeviceServices.ContextMenuItems
        {...props}
        configureLayout={Device.CONFIGURE_LAYOUT}
        taskContextMenuItemConfigs={TASK_CONTEXT_MENU_ITEM_CONFIGS}
      />
      {!allPassive && (
        <Menu.Item itemKey="ethercat.setPassive" onClick={handleSetPassive}>
          <Icon.Disable />
          Set passive
        </Menu.Item>
      )}
      {!allActive && (
        <Menu.Item itemKey="ethercat.setActive" onClick={handleSetActive}>
          <Icon.Enable />
          Set active
        </Menu.Item>
      )}
    </>
  );
};
