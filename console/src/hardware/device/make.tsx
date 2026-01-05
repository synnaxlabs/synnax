// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";
import { z } from "zod";

import { LabJack } from "@/hardware/labjack";
import { Modbus } from "@/hardware/modbus";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { type Layout } from "@/layout";
import { type Ontology } from "@/ontology";

export const makeZ = z.enum([
  LabJack.Device.MAKE,
  Modbus.Device.MAKE,
  NI.Device.MAKE,
  OPC.Device.MAKE,
]);
export type Make = z.infer<typeof makeZ>;

export const getMake = (make: unknown): Make | null =>
  makeZ.safeParse(make).data ?? null;

export const getIconString = (make: Make | null): string => {
  switch (make) {
    case LabJack.Device.MAKE:
      return "Logo.LabJack";
    case Modbus.Device.MAKE:
      return "Logo.Modbus";
    case NI.Device.MAKE:
      return "Logo.NI";
    case OPC.Device.MAKE:
      return "Logo.OPC";
    default:
      return "Device";
  }
};

export const hasIdentifier = (make: Make | null): boolean =>
  make === LabJack.Device.MAKE || make === NI.Device.MAKE;

const MAKE_ICONS: Record<Make, Icon.ReactElement> = {
  [LabJack.Device.MAKE]: <Icon.Logo.LabJack />,
  [Modbus.Device.MAKE]: <Icon.Logo.Modbus />,
  [NI.Device.MAKE]: <Icon.Logo.NI />,
  [OPC.Device.MAKE]: <Icon.Logo.OPC />,
};

export const getIcon = (make: Make | null) =>
  make ? MAKE_ICONS[make] : <Icon.Device />;

export const CONFIGURE_LAYOUTS: Record<Make, Layout.BaseState> = {
  [LabJack.Device.MAKE]: LabJack.Device.CONFIGURE_LAYOUT,
  [Modbus.Device.MAKE]: Modbus.Device.CONNECT_LAYOUT,
  [NI.Device.MAKE]: NI.Device.CONFIGURE_LAYOUT,
  [OPC.Device.MAKE]: OPC.Device.CONNECT_LAYOUT,
};

const CONTEXT_MENU_ITEMS: Record<
  Make,
  (props: Ontology.TreeContextMenuProps) => ReactElement | null
> = {
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
