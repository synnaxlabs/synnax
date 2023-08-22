// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { AiFillApi } from "react-icons/ai";

import { connectClusterWindowLayout } from "@/cluster/components";
import { Command } from "@/palette/Palette";

export const connectClusterCommand: Command = {
  key: "connect-cluster",
  name: "Connect a Cluster",
  icon: <AiFillApi />,
  onSelect: ({ placeLayout: layoutPlacer }) => layoutPlacer(connectClusterWindowLayout),
};

export const CLUSTER_COMMANDS = [connectClusterCommand];
