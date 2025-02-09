// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Sequence } from "@/hardware/task/sequence";
import { Layout } from "@/layout";

export const SELECTABLES: Layout.Selectable[] = [
  ...LabJack.Task.SELECTABLES,
  ...NI.Task.SELECTABLES,
  ...OPC.Task.SELECTABLES,
  ...Sequence.SELECTABLES,
];

export const SELECTOR_LAYOUT_TYPE = "taskSelector";

export const ZERO_SELECTOR_LAYOUT: Layout.BaseState = {
  type: SELECTOR_LAYOUT_TYPE,
  icon: "Task",
  location: "mosaic",
  name: "New Task",
  key: SELECTOR_LAYOUT_TYPE,
};

const SelectorComponent = Layout.createSelectorComponent(SELECTABLES);

export const Selector = ({
  text = "Select a Task Type",
  ...rest
}: Layout.SelectorProps) => <SelectorComponent text={text} {...rest} />;
