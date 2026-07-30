// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { type ExecutionContext } from "@/components/arc/types";
import {
  type TabEntry,
  Tabs as Base,
  type TabsProps as BaseProps,
} from "@/components/tabs/Tabs";

interface Entry extends TabEntry {
  tabKey: ExecutionContext;
}

const TABS: Entry[] = [
  { tabKey: "flow", name: "Flow", icon: <Icon.ArcFlow /> },
  { tabKey: "function", name: "Func", icon: <Icon.ArcFunc /> },
];

export type TabsProps = Omit<BaseProps, "tabs" | "queryParamKey">;

export const Tabs = (props: TabsProps) => (
  <Base queryParamKey="context" tabs={TABS} {...props} />
);
