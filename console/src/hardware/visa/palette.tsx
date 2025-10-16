// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { type Command } from "@/palette";

export const connectCommand: Command = {
  key: "visa-connect",
  name: "Connect to VISA Instrument",
  icon: <Icon.Logo.VISA />,
  onSelect: ({ placeLayout }) =>
    placeLayout({
      key: "configureVISADevice",
      type: "configureVISADevice",
      name: "Connect to VISA Instrument",
      location: "modal",
      window: {
        resizable: false,
        size: { height: 550, width: 650 },
        navTop: true,
      },
    }),
};

export const COMMANDS = [connectCommand];