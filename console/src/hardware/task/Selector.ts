// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { LabJack } from "@/hardware/labjack";
import { Modbus } from "@/hardware/modbus";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Sequence } from "@/hardware/task/sequence";
import { type Layout } from "@/layout";
import { Selector as CoreSelector } from "@/selector";

export const SELECTABLES: CoreSelector.Selectable[] = [
  ...LabJack.Task.SELECTABLES,
  ...Modbus.Task.SELECTABLES,
  ...NI.Task.SELECTABLES,
  ...OPC.Task.SELECTABLES,
  ...Sequence.SELECTABLES,
].map((selectable) => ({
  ...selectable,
  useVisible: () => Access.useEditGranted(task.TYPE_ONTOLOGY_ID),
}));

export const SELECTOR_LAYOUT_TYPE = "taskSelector";

export const SELECTOR_LAYOUT: Layout.BaseState = {
  type: SELECTOR_LAYOUT_TYPE,
  icon: "Task",
  location: "mosaic",
  name: "New Task",
};

export const Selector = CoreSelector.createSelector(SELECTABLES, "Select a Task Type");
