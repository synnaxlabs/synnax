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

import { CONNECT_LAYOUT } from "@/hardware/sift/device/Connect";
import { Palette } from "@/palette";

const useVisible = () => Access.useCreateGranted(device.TYPE_ONTOLOGY_ID);

export const ConnectSiftDeviceCommand = Palette.createSimpleCommand({
  key: "sift-connect-device",
  name: "Connect to Sift",
  icon: <Icon.Logo.Sift />,
  layout: CONNECT_LAYOUT,
  useVisible,
});

export const COMMANDS = [ConnectSiftDeviceCommand];
