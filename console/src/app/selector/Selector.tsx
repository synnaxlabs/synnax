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
import { useCallback } from "react";

import { SELECTABLES } from "@/app/selector/selectables";
import { Panel } from "@/platform/panel";
import { Selector as Base } from "@/platform/selector";

export const useVisible = (): boolean =>
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

export const useOpenTab = (): (() => void) => {
  const openTab = Panel.useOpenTab();
  return useCallback(() => openTab(createEmptyTab()), [openTab]);
};
