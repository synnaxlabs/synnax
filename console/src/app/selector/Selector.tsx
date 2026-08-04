// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { SELECTABLES } from "@/app/selector/selectables";
import { type Panel } from "@/platform/panel";
import { Selector as Base } from "@/platform/selector";

export const useVisible = (): boolean =>
  // It's safe to call hooks in map since SELECTABLES is a module-level constant
  // and never changes between renders, ensuring consistent hook order.
  SELECTABLES.map((s) => s.useVisible?.() ?? true).some(Boolean);

export const Selector = Base.create({
  selectables: SELECTABLES,
  icon: <Icon.Component />,
  tabTitle: "Create component",
  text: "Create a component",
});

export const TABS: Panel.Tabs = { [Base.TAB_TYPE]: Selector };
