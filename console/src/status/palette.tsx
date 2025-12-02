// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Access, Icon, Status } from "@synnaxlabs/pluto";

import { type Palette } from "@/palette";
import { CREATE_LAYOUT } from "@/status/Create";
import { EXPLORER_LAYOUT } from "@/status/Explorer";

export const CREATE_COMMAND: Palette.Command = {
  key: "create_status",
  name: "Create a Status",
  icon: <Status.CreateIcon />,
  onSelect: ({ placeLayout }) => placeLayout(CREATE_LAYOUT),
  visible: ({ store, client }) =>
    Access.editGranted({ id: status.ontologyID(""), store, client }),
};

export const OPEN_EXPLORER_COMMAND: Palette.Command = {
  key: "open_status_explorer",
  name: "Open Status Explorer",
  icon: <Icon.Explore />,
  onSelect: ({ placeLayout }) => placeLayout(EXPLORER_LAYOUT),
  visible: ({ store, client }) =>
    Access.viewGranted({ id: status.ontologyID(""), store, client }),
};

export const COMMANDS = [CREATE_COMMAND, OPEN_EXPLORER_COMMAND];
