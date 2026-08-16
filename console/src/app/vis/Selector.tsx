// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Selector as AppSelector } from "@/app/selector";
import { Selector as Base } from "@/platform/selector";

export const useSelectorVisible = (): boolean =>
  // It's safe to call hooks in map since VIS_SELECTABLES is a module-level constant
  // and never changes between renders, ensuring consistent hook order.
  AppSelector.VIS_SELECTABLES.map((s) => s.useVisible?.() ?? true).some(Boolean);

export const Selector = Base.create({
  selectables: AppSelector.VIS_SELECTABLES,
  icon: <Icon.Visualize />,
  tabTitle: "Create visualization",
  text: "Create a visualization",
});
