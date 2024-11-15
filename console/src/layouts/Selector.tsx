// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { v4 as uuid } from "uuid";

import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Schematic } from "@/schematic";
import { Table } from "@/table";

const SELECTABLES: Layout.Selectable[] = [
  LinePlot.SELECTABLE,
  Schematic.SELECTABLE,
  Table.SELECTABLE,
  ...Log.SELECTABLES,
  ...LabJack.SELECTABLES,
  ...NI.SELECTABLES,
  ...OPC.SELECTABLES,
];

export const SELECTOR_TYPE = "visLayoutSelector";

export const createSelector = (
  props: Omit<Partial<Layout.State>, "type">,
): Omit<Layout.State, "windowKey"> => {
  const { location = "mosaic", name = "New Layout", key = uuid(), window, tab } = props;
  return {
    type: SELECTOR_TYPE,
    location,
    name,
    key,
    window,
    tab,
  };
};

export const Selector = (props: Layout.SelectorProps): ReactElement => {
  const canCreateSchematic = Schematic.useSelectHasPermission();
  const selectables = SELECTABLES.filter((s) => {
    if (s.key === Schematic.SELECTABLE.key) return canCreateSchematic;
    return true;
  });
  return Layout.createSelectorComponent(selectables)(props);
};
