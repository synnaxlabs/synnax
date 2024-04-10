// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { connectWindowLayout } from "@/hardware/opcua/Configure";
import { type Command } from "@/palette/Palette";

export const createCommand: Command = {
  key: "opcua-connect-server",
  name: "OPCUA: Connect a New Server",
  icon: <Icon.PID />,
  onSelect: ({ placeLayout }) => placeLayout(connectWindowLayout),
};

export const COMMANDS = [createCommand];
