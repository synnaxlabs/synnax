// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { AiFillApi } from "react-icons/ai";

import { Cluster } from "@/cluster";
import { Layout } from "@/layout";
import { type Palette } from "@/palette";
import { Runtime } from "@/runtime";
import { Workspace } from "@/workspace";

const CONNECT_COMMAND: Palette.Command = {
  key: "connect-cluster",
  name: "Add a Core",
  icon: <AiFillApi />,
  onSelect: ({ placeLayout }) => placeLayout(Cluster.CONNECT_LAYOUT),
  visible: () => Runtime.ENGINE === "tauri",
};

const LOGOUT_COMMAND: Palette.Command = {
  key: "logout",
  name: "Log Out",
  icon: <Icon.Logout />,
  onSelect: ({ store }) => {
    store.dispatch(Cluster.setActive(null));
    store.dispatch(Workspace.setActive(null));
    store.dispatch(Layout.clearWorkspace());
  },
};

export const COMMANDS = [CONNECT_COMMAND, LOGOUT_COMMAND];
