// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";

import { Selector as AppSelector } from "@/app/selector";
import { type Layout } from "@/platform/layout";
import { Selector as Base } from "@/platform/selector";
import { type Session } from "@/session";

export const SELECTOR_LAYOUT_TYPE = "visualizationSelector";

export const useSelectorVisible = (): boolean =>
  // It's safe to call hooks in map since SELECTABLES is a module-level constant
  // and never changes between renders, ensuring consistent hook order.
  AppSelector.VIS_SELECTABLES.map((s) => s.useVisible?.() ?? true).some(Boolean);

export const createSelectorLayout = (): Session.Layout.BaseState => ({
  type: SELECTOR_LAYOUT_TYPE,
  icon: "Visualize",
  location: "mosaic",
  name: "New Visualization",
  key: uuid.create(),
});

export const Selector: Layout.Renderer = (props) => (
  <Base.Selector
    selectables={AppSelector.VIS_SELECTABLES}
    text="Select a Visualization Type"
    {...props}
  />
);
