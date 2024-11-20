// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Layout } from "@/layout";

export const SELECTABLES: Layout.Selectable[] = [
  ...LabJack.SELECTABLES,
  ...NI.SELECTABLES,
  ...OPC.SELECTABLES,
];

export const SELECTOR_TYPE = "taskSelector";

export const createSelector = ({
  location = "mosaic",
  name = "New Task",
  key = id.id(),
  ...props
}: Omit<Partial<Layout.State>, "type" | "icon">): Omit<Layout.State, "windowKey"> => ({
  type: SELECTOR_TYPE,
  icon: "Task",
  location,
  name,
  key,
  ...props,
});

const SELECTOR = Layout.createSelectorComponent(SELECTABLES);

export const Selector = ({
  text = "Select a Task Type",
  ...props
}: Layout.SelectorProps): ReactElement => SELECTOR({ text, ...props });
