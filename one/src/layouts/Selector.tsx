// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { v4 as uuidv4 } from "uuid";

import { Code } from "@/code";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Schematic } from "@/schematic";

const SELECTABLES: Layout.Selectable[] = [
  LinePlot.SELECTABLE,
  Schematic.SELECTABLE,
  Code.EDITOR_SELECTABLE,
  ...NI.SELECTABLES,
  ...OPC.SELECTABLES,
];

export const SELECTOR_TYPE = "visLayoutSelector";

export const createSelector = (
  props: Omit<Partial<Layout.State>, "type">,
): Omit<Layout.State, "windowKey"> => {
  const {
    location = "mosaic",
    name = "New Layout",
    key = uuidv4(),
    window,
    tab,
  } = props;
  return {
    type: SELECTOR_TYPE,
    location,
    name,
    key,
    window,
    tab,
  };
};

export const Selector = Layout.createSelectorComponent(SELECTABLES);