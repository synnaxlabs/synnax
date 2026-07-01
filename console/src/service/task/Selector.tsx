// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { Selector as BaseSelector } from "@/component/selector";
import { EtherCAT } from "@/service/ethercat";
import { HTTP } from "@/service/http";
import { LabJack } from "@/service/labjack";
import { Modbus } from "@/service/modbus";
import { NI } from "@/service/ni";
import { OPC } from "@/service/opc";
import { PagerDuty } from "@/service/pagerduty";
import { type Session } from "@/session";

const withTaskVisibility = (
  Selectable: BaseSelector.Selectable,
): BaseSelector.Selectable => {
  const WrappedSelectable: BaseSelector.Selectable = (props) => {
    const hasCreatePermission = Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);
    if (!hasCreatePermission) return null;
    return <Selectable {...props} />;
  };
  WrappedSelectable.type = Selectable.type;
  return WrappedSelectable;
};

export const SELECTABLES: BaseSelector.Selectable[] = [
  ...EtherCAT.Task.SELECTABLES,
  ...HTTP.Task.SELECTABLES,
  ...LabJack.Task.SELECTABLES,
  ...Modbus.Task.SELECTABLES,
  ...NI.Task.SELECTABLES,
  ...OPC.Task.SELECTABLES,
  ...PagerDuty.Task.SELECTABLES,
].map(withTaskVisibility);

export const SELECTOR_LAYOUT_TYPE = "taskSelector";

export const SELECTOR_LAYOUT: Session.Layout.BaseState = {
  type: SELECTOR_LAYOUT_TYPE,
  icon: "Task",
  location: "mosaic",
  name: "New Task",
};

export const Selector = BaseSelector.createSelector(SELECTABLES, "Select a Task Type");
