// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { z } from "zod";

import { EtherCAT } from "@/hardware/ethercat";
import { HTTP } from "@/hardware/http";
import { LabJack } from "@/hardware/labjack";
import { Modbus } from "@/hardware/modbus";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { type Ontology } from "@/ontology";

export const makeZ = z.enum([
  EtherCAT.Device.MAKE,
  HTTP.Device.MAKE,
  LabJack.Device.MAKE,
  Modbus.Device.MAKE,
  NI.Device.MAKE,
  OPC.Device.MAKE,
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
  [OPC.Device.MAKE]: <Icon.Logo.OPC />,
};

export const getIcon = (make: Make | null) =>
  make ? MAKE_ICONS[make] : <Icon.Device />;

/**
 * useConfigureModal returns an opener that launches the configure/connect modal for a device
 * of the given make. Every integration's modal hook is called unconditionally so the
 * returned opener can dispatch to any make at call time without violating the rules of
 * hooks.
 */
export const useConfigureModal = (): ((
  make: Make,
  deviceKey?: string,
  title?: string,
) => void) => {
  const ethercat = EtherCAT.Device.useConfigureModal();
  const http = HTTP.Device.useConnectModal();
  const labjack = LabJack.Device.useConfigureModal();
  const modbus = Modbus.Device.useConnectModal();
  const ni = NI.Device.useConfigureModal();
  const opc = OPC.Device.useConnectModal();
  return useCallback(
    (make, deviceKey, title) => {
      const openers: Record<
        Make,
        (args: { deviceKey?: string; title?: string }) => void
      > = {
        [EtherCAT.Device.MAKE]: ethercat,
        [HTTP.Device.MAKE]: http,
        [LabJack.Device.MAKE]: labjack,
        [Modbus.Device.MAKE]: modbus,
        [NI.Device.MAKE]: ni,
        [OPC.Device.MAKE]: opc,
      };
      openers[make]({ deviceKey, title });
    },
    [ethercat, http, labjack, modbus, ni, opc],
  );
};

const CONTEXT_MENU_ITEMS: Partial<
  Record<Make, (props: Ontology.TreeContextMenuProps) => ReactElement | null>
> = {
  [EtherCAT.Device.MAKE]: EtherCAT.DeviceServices.ContextMenuItems,
  [HTTP.Device.MAKE]: HTTP.DeviceServices.ContextMenuItems,
  [LabJack.Device.MAKE]: LabJack.DeviceServices.ContextMenuItems,
  [Modbus.Device.MAKE]: Modbus.DeviceServices.ContextMenuItems,
  [NI.Device.MAKE]: NI.DeviceServices.ContextMenuItems,
  [OPC.Device.MAKE]: OPC.DeviceServices.ContextMenuItems,
};

export const getContextMenuItems = (make: unknown) => {
  const m = getMake(make);
  if (m == null) return null;
  return CONTEXT_MENU_ITEMS[m];
};
