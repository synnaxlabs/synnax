// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";

import { Arc } from "@/feature/arc";
import { Task } from "@/feature/task";
import { Vis } from "@/feature/vis";
import { type Layout } from "@/primitive/layout";
import { Selector as BaseSelector } from "@/primitive/selector";
import { type Session } from "@/session";

const SELECTABLES: BaseSelector.Selectable[] = [
  ...Vis.SELECTABLES,
  ...Task.SELECTABLES,
  ...Arc.SELECTABLES,
];

export const useSelectorVisible = (): boolean =>
  // It's safe to call hooks in map since SELECTABLES is a module-level constant
  // and never changes between renders, ensuring consistent hook order.
  SELECTABLES.map((s) => s.useVisible?.() ?? true).some(Boolean);

export const SELECTOR_LAYOUT_TYPE = "layoutSelector";

export interface CreateSelectorLayoutArgs extends Omit<
  Session.Layout.BaseState,
  "type" | "icon" | "location" | "name" | "key"
> {}

export const createSelectorLayout = (
  args: CreateSelectorLayoutArgs = {},
): Session.Layout.BaseState => ({
  ...args,
  type: SELECTOR_LAYOUT_TYPE,
  icon: "Visualize",
  location: "mosaic",
  name: "New Component",
  key: uuid.create(),
});

export const Selector: Layout.Renderer = (props) => (
  <BaseSelector.Selector
    selectables={SELECTABLES}
    text="Select a Component Type"
    {...props}
  />
);
