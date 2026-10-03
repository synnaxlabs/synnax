// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { type Platform } from "@/components/platform/choice";
import { INFO } from "@/components/platform/Platform";
import { type FilterProps, type TabEntry, Tabs as Base } from "@/components/tabs/Tabs";

type Target = Exclude<Platform, "Docker"> | "ni-linux-rt";

const TABS: TabEntry<Target>[] = [
  { tabKey: "ni-linux-rt", name: "NI Linux Real-Time", icon: <Icon.Logo.NI /> },
  ...INFO.flatMap(({ key, ...p }) => (key === "Docker" ? [] : [{ ...p, tabKey: key }])),
];

export type TabsProps = FilterProps<Target>;

export const Tabs = (props: TabsProps) => (
  <Base queryParamKey="platform" tabs={TABS} {...props} />
);
