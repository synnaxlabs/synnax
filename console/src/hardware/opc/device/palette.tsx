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

import { CONNECT_LAYOUT } from "@/hardware/opc/device/Connect";
import { Palette } from "@/palette";

const useVisible = () => Access.useUpdateGranted(device.TYPE_ONTOLOGY_ID);

export const ConnectOPCServerCommand = Palette.createSimpleCommand({
  key: "opc-ua-connect-server",
  name: "Connect an OPC UA server",
  icon: <Icon.Logo.OPC />,
  layout: CONNECT_LAYOUT,
  useVisible,
});

export const COMMANDS = [ConnectOPCServerCommand];
