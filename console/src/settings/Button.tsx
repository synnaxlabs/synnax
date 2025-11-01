// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Access, Button as Core, Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Layout } from "@/layout";
import { CLUSTER_SETTINGS_LAYOUT } from "@/settings/ClusterSettings";

export const Button = (): ReactElement | null => {
  const isAdmin = Access.useIsAdmin();
  const placeLayout = Layout.usePlacer();
  if (!isAdmin) return null;
  return (
    <Core.Button
      onClick={() => placeLayout(CLUSTER_SETTINGS_LAYOUT)}
      tooltip="Cluster Settings"
      tooltipLocation="right"
      size="large"
      variant="outlined"
    >
      <Icon.Settings />
    </Core.Button>
  );
};
