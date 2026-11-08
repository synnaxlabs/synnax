// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { Platform } from "@/components/platform";
import { Tabs as Base } from "@/components/tabs";

type Target = Exclude<Platform.Platform, "Docker"> | "ni-linux-rt";

const TABS: Base.TabEntry<Target>[] = [
  { tabKey: "ni-linux-rt", name: "NI Linux Real-Time", icon: <Icon.Logo.NI /> },
  ...Platform.INFO.flatMap(({ key, ...p }) =>
    key === "Docker" ? [] : [{ ...p, tabKey: key }],
  ),
];

export type TabsProps = Base.FilterProps<Target>;

export const Tabs = (props: TabsProps) => (
  <Base.Tabs queryParamKey="platform" tabs={TABS} {...props} />
);
