// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { AiFillApi } from "react-icons/ai";

import { connectWindowLayout } from "@/cluster/Connect";
import type { Command } from "@/palette/Palette";

export const connectCommand: Command = {
  key: "connect-cluster",
  name: "Connect a Cluster",
  icon: <AiFillApi />,
  onSelect: ({ placeLayout }) => placeLayout(connectWindowLayout),
};

export const COMMANDS = [connectCommand];
