// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { type Platform, PLATFORMS } from "@/components/platform/Platform";
import { Tabs as Base, type TabsProps as BaseProps } from "@/components/tabs/Tabs";

const TABS = PLATFORMS.filter(
  ({ key }) => key === "Linux" || key === "Windows" || key === "macOS",
).map(({ key, ...p }) => ({
  ...p,
  tabKey: key,
}));
TABS.unshift({
  tabKey: "ni-linux-rt" as Platform,
  name: "NI Linux RT",
  icon: <Icon.Logo.NI />,
});

export interface TabsProps extends Omit<BaseProps, "tabs" | "queryParamKey"> {
  exclude?: Platform[];
  priority?: Platform[];
}

export const Tabs = (props: TabsProps) => (
  <Base queryParamKey="platform" tabs={TABS} {...props} />
);
