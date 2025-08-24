// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { detectServingCluster } from "@/cluster/autoConnect";
import { Dropdown } from "@/cluster/Dropdown";
import { useSelect } from "@/cluster/selectors";

import { ConnectionBadge, ConnectionStatusBadge } from "./Badges";

/**
 * ConnectionStatus displays either the cluster dropdown for normal console mode,
 * or a simple connection status when served by a cluster.
 */
export const ConnectionStatus = (): ReactElement => {
  const servingCluster = detectServingCluster();
  if (servingCluster == null) return <Dropdown />;
  return <ConnectionBadge />;
};
