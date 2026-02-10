// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";

import { CONNECT_LAYOUT } from "@/hardware/modbus/device/Connect";
import { Palette } from "@/palette";

const useVisible = () => Access.useUpdateGranted(device.TYPE_ONTOLOGY_ID);

export const ConnectModbusServerCommand = Palette.createSimpleCommand({
  key: "modbus-connect-server",
  name: "Connect a Modbus server",
  icon: <Icon.Logo.Modbus />,
  layout: CONNECT_LAYOUT,
  useVisible,
});

export const COMMANDS = [ConnectModbusServerCommand];
