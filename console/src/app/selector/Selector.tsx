// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";

import { Vis } from "@/app/vis";
import { Arc } from "@/feature/arc";
import { Task } from "@/feature/task";
import { Panel } from "@/platform/panel";
import { Selector as Base } from "@/platform/selector";

const SELECTABLES: Base.Selectable[] = [
  ...Vis.SELECTABLES,
  ...Task.SELECTABLES,
  ...Arc.SELECTABLES,
];

export const useSelectorVisible = (): boolean =>
  // It's safe to call hooks in map since SELECTABLES is a module-level constant
  // and never changes between renders, ensuring consistent hook order.
  SELECTABLES.map((s) => s.useVisible?.() ?? true).some(Boolean);

export const Selector = Base.create({
  selectables: SELECTABLES,
  icon: <Icon.Add />,
  tabTitle: "Create component",
  text: "Create a component",
});

export const TAB_TYPE = "selector";

export const TABS: Panel.Tabs = { [TAB_TYPE]: Selector };

export const createEmptyTab = (): panel.NewTab => ({
  variant: "view",
  type: TAB_TYPE,
  args: {},
});

export type PickerVariant = "component" | "task";

export const useOpenTab = (): ((variant?: PickerVariant) => void) => {
  const openTab = Panel.useOpenTab();
  return (variant: PickerVariant = "component") =>
    openTab({ variant: "view", type: TAB_TYPE, args: { variant } });
};
