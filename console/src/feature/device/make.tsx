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
import { type ReactElement, useCallback } from "react";
import { z } from "zod";

import { ARINC429 } from "@/feature/arinc429";
import { CAN } from "@/feature/can";
import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { LabJack } from "@/feature/labjack";
import { MIL1553 } from "@/feature/mil1553";
import { Modbus } from "@/feature/modbus";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { Serial } from "@/feature/serial";
import { TCP } from "@/feature/tcp";
import { UDP } from "@/feature/udp";
import { FLAGS } from "@/flags";
import { type Tree } from "@/platform/tree";

const BUS_MAKES = [
  CAN.Device.MAKE,
  Serial.Device.MAKE,
  TCP.Device.MAKE,
  UDP.Device.MAKE,
  ARINC429.Device.MAKE,
  MIL1553.Device.MAKE,
] as const;

export const makeZ = z.enum([
  ...BUS_MAKES,
  EtherCAT.Device.MAKE,
  HTTP.Device.MAKE,
  LabJack.Device.MAKE,
  Modbus.Device.MAKE,
  NI.Device.MAKE,
  OPCUA.Device.MAKE,
]);
export type Make = z.infer<typeof makeZ>;

const BUS_MAKE_SET = new Set<Make>(BUS_MAKES);

export const getMake = (make: unknown): Make | null => {
  const parsed = makeZ.safeParse(make).data ?? null;
  if (parsed != null && !FLAGS.can && BUS_MAKE_SET.has(parsed)) return null;
  return parsed;
};

const MAKE_ICONS: Record<Make, Icon.ReactElement> = {
  [CAN.Device.MAKE]: <Icon.Hardware />,
  [Serial.Device.MAKE]: <Icon.Connect />,
  [TCP.Device.MAKE]: <Icon.Link />,
  [UDP.Device.MAKE]: <Icon.Bridge />,
  [ARINC429.Device.MAKE]: <Icon.Wave.Square />,
  [MIL1553.Device.MAKE]: <Icon.Node />,
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
  const can = CAN.Device.useConnectModal();
  const serial = Serial.Device.useConnectModal();
  const tcp = TCP.Device.useConnectModal();
  const udp = UDP.Device.useConnectModal();
  const arinc429 = ARINC429.Device.useConnectModal();
  const mil1553 = MIL1553.Device.useConnectModal();
  const ethercat = EtherCAT.Device.useConfigureModal();
  const http = HTTP.Device.useConnectModal();
  const labjack = LabJack.Device.useConfigureModal();
  const modbus = Modbus.Device.useConnectModal();
  const ni = NI.Device.useConfigureModal();
  const opcua = OPCUA.Device.useConnectModal();
  return useCallback(
    (make, deviceKey) => {
      const openers: Record<Make, (args: { deviceKey: device.Key }) => void> = {
        [CAN.Device.MAKE]: can,
        [Serial.Device.MAKE]: serial,
        [TCP.Device.MAKE]: tcp,
        [UDP.Device.MAKE]: udp,
        [ARINC429.Device.MAKE]: arinc429,
        [MIL1553.Device.MAKE]: mil1553,
        [EtherCAT.Device.MAKE]: ethercat,
        [HTTP.Device.MAKE]: http,
        [LabJack.Device.MAKE]: labjack,
        [Modbus.Device.MAKE]: modbus,
        [NI.Device.MAKE]: ni,
        [OPCUA.Device.MAKE]: opcua,
      };
      openers[make]({ deviceKey });
    },
    [
      can,
      serial,
      tcp,
      udp,
      arinc429,
      mil1553,
      ethercat,
      http,
      labjack,
      modbus,
      ni,
      opcua,
    ],
  );
};

const CONTEXT_MENU_ITEMS: Partial<
  Record<Make, (props: Tree.ContextMenuProps) => ReactElement | null>
> = {
  [CAN.Device.MAKE]: CAN.Device.ContextMenuItems,
  [Serial.Device.MAKE]: Serial.Device.ContextMenuItems,
  [TCP.Device.MAKE]: TCP.Device.ContextMenuItems,
  [UDP.Device.MAKE]: UDP.Device.ContextMenuItems,
  [ARINC429.Device.MAKE]: ARINC429.Device.ContextMenuItems,
  [MIL1553.Device.MAKE]: MIL1553.Device.ContextMenuItems,
  [EtherCAT.Device.MAKE]: EtherCAT.Device.ContextMenuItems,
  [HTTP.Device.MAKE]: HTTP.Device.ContextMenuItems,
  [LabJack.Device.MAKE]: LabJack.Device.ContextMenuItems,
  [Modbus.Device.MAKE]: Modbus.Device.ContextMenuItems,
  [NI.Device.MAKE]: NI.Device.ContextMenuItems,
  [OPCUA.Device.MAKE]: OPCUA.Device.ContextMenuItems,
};

export const getContextMenuItems = (make: unknown) => {
  const m = getMake(make);
  if (m == null) return null;
  return CONTEXT_MENU_ITEMS[m];
};
