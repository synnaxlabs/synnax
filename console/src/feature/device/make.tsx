// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ComponentType } from "react";
import { z } from "zod";

import { useConfigureModal as useConfigureHardwareModal } from "@/feature/device/Configure";
import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { type Device as PlatformDevice } from "@/platform/device";
import { type Tree } from "@/platform/tree";

export const makeZ = z.enum([
  EtherCAT.Device.MAKE,
  HTTP.Device.MAKE,
  LabJack.Device.MAKE,
  Modbus.Device.MAKE,
  NI.Device.MAKE,
  OPCUA.Device.MAKE,
]);
export type Make = z.infer<typeof makeZ>;

export const getMake = (make: unknown): Make | null =>
  makeZ.safeParse(make).data ?? null;

const MAKE_ICONS: Record<Make, Icon.ReactElement> = {
  [EtherCAT.Device.MAKE]: <Icon.Logo.EtherCAT />,
  [HTTP.Device.MAKE]: <Icon.Logo.HTTP />,
  [LabJack.Device.MAKE]: <Icon.Logo.LabJack />,
  [Modbus.Device.MAKE]: <Icon.Logo.Modbus />,
  [NI.Device.MAKE]: <Icon.Logo.NI />,
  [OPCUA.Device.MAKE]: <Icon.Logo.OPCUA />,
};

export const getIcon = (make: Make | null) =>
  make ? MAKE_ICONS[make] : <Icon.Device />;

/**
 * useConfigureModal returns an opener that launches the configure/connect modal for a device
 * of the given make. Every integration's modal hook is called unconditionally so the
 * returned opener can dispatch to any make at call time without violating the rules of
 * hooks.
 */
export const useConfigureModal = (): ((make: Make, deviceKey: device.Key) => void) => {
  const configure = useConfigureHardwareModal();
  const http = HTTP.Device.useConnectModal();
  const modbus = Modbus.Device.useConnectModal();
  const opcua = OPCUA.Device.useConnectModal();
  return (make, deviceKey) => {
    const openers: Record<Make, (args: { deviceKey: device.Key }) => void> = {
      [EtherCAT.Device.MAKE]: ({ deviceKey }) =>
        configure({ deviceKey, integration: EtherCAT.Device.INTEGRATION }),
      [HTTP.Device.MAKE]: ({ deviceKey }) => void http({ deviceKey }),
      [LabJack.Device.MAKE]: ({ deviceKey }) =>
        configure({ deviceKey, integration: LabJack.Device.INTEGRATION }),
      [Modbus.Device.MAKE]: ({ deviceKey }) => void modbus({ deviceKey }),
      [NI.Device.MAKE]: ({ deviceKey }) =>
        configure({ deviceKey, integration: NI.Device.INTEGRATION }),
      [OPCUA.Device.MAKE]: ({ deviceKey }) => void opcua({ deviceKey }),
    };
    openers[make]({ deviceKey });
  };
};

const CONTEXT_MENU_ITEMS: Record<
  Make,
  ComponentType<PlatformDevice.ContextMenuItemsProps>
> = {
  [EtherCAT.Device.MAKE]: EtherCAT.Device.ContextMenuItems,
  [HTTP.Device.MAKE]: HTTP.ContextMenuItems,
  [LabJack.Device.MAKE]: LabJack.Device.ContextMenuItems,
  [Modbus.Device.MAKE]: Modbus.ContextMenuItems,
  [NI.Device.MAKE]: NI.Device.ContextMenuItems,
  [OPCUA.Device.MAKE]: OPCUA.ContextMenuItems,
};

export interface ContextMenuItemsProps extends Tree.ContextMenuProps {
  make: Make;
}

/** The context menu items of a device's integration, wired to its configure form. */
export const ContextMenuItems = ({ make, ...rest }: ContextMenuItemsProps) => {
  const configure = useConfigureModal();
  const Items = CONTEXT_MENU_ITEMS[make];
  return <Items {...rest} onConfigure={(deviceKey) => configure(make, deviceKey)} />;
};
