// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { type ExecutionContext } from "@/components/arc/types";
import { type FilterProps, type TabEntry, Tabs as Base } from "@/components/tabs/Tabs";

const TABS: TabEntry<ExecutionContext>[] = [
  { tabKey: "flow", name: "Flow", icon: <Icon.ArcFlow /> },
  { tabKey: "function", name: "Func", icon: <Icon.ArcFunc /> },
];

export type TabsProps = FilterProps<ExecutionContext>;

export const Tabs = (props: TabsProps) => (
  <Base queryParamKey="context" tabs={TABS} {...props} />
);
