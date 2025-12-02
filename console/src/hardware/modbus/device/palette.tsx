// Copyright 2025 Synnax Labs, Inc.
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
import { type Palette } from "@/palette";

const CONNECT_SERVER_COMMAND: Palette.Command = {
  key: "modbus-connect-server",
  name: "Connect a Modbus Server",
  icon: <Icon.Logo.Modbus />,
  onSelect: ({ placeLayout }) => placeLayout(CONNECT_LAYOUT),
  visible: ({ store, client }) => Access.editGranted({ id: device.ontologyID(""), store, client }),
};

export const COMMANDS: Palette.Command[] = [CONNECT_SERVER_COMMAND];
