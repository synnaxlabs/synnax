// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tabs } from "@synnaxlabs/pluto";
import { useCallback, useState } from "react";

import { type Layout } from "@/layout";
import { ClusterTab } from "@/settings/ClusterTab";
import { UsersTab } from "@/settings/users/UsersTab";

const SETTINGS_TABS: Tabs.Tab[] = [
  { tabKey: "users", name: "Users" },
  { tabKey: "roles", name: "Roles" },
  { tabKey: "policies", name: "Policies" },
  { tabKey: "cluster", name: "Cluster" },
];

export const ClusterSettings: Layout.Renderer = () => {
  const [activeTab, setActiveTab] = useState("users");

  const content = useCallback(({ tabKey }: Tabs.Tab) => {
    switch (tabKey) {
      case "users":
        return <UsersTab />;
      case "roles":
        return <div>Roles Tab - Coming Soon</div>;
      case "policies":
        return <div>Policies Tab - Coming Soon</div>;
      case "cluster":
        return <ClusterTab />;
      default:
        return null;
    }
  }, []);

  return (
    <Tabs.Tabs selected={activeTab} onSelect={setActiveTab} tabs={SETTINGS_TABS}>
      {content}
    </Tabs.Tabs>
  );
};

export const CLUSTER_SETTINGS_LAYOUT: Layout.BaseState = {
  key: "cluster-settings",
  type: "clusterSettings",
  name: "Cluster Settings",
  icon: "Settings",
  location: "modal",
  window: {
    resizable: true,
    size: { height: 600, width: 900 },
    navTop: true,
  },
};
