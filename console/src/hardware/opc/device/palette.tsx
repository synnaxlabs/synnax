// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Icon as PIcon } from "@synnaxlabs/pluto";

import { createConfigureLayout } from "@/hardware/opc/device/Configure";
import { type Palette } from "@/palette";

const CONNECT_SERVER_COMMAND: Palette.Command = {
  key: "opc-connect-server",
  name: "Connect an OPC UA Server",
  icon: (
    <PIcon.Create>
      <Icon.Logo.OPC />
    </PIcon.Create>
  ),
  onSelect: ({ placeLayout }) => placeLayout(createConfigureLayout()),
};

export const COMMANDS: Palette.Command[] = [CONNECT_SERVER_COMMAND];
