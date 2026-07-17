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

import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { NI } from "@/feature/ni";
import { OPC } from "@/feature/opc";
import { PagerDuty } from "@/feature/pagerduty";
import { Slack } from "@/feature/slack";
import { Selector as Base } from "@/platform/selector";

const withTaskVisibility = (Selectable: Base.Selectable): Base.Selectable => {
  const WrappedSelectable: Base.Selectable = (props) => {
    const hasCreatePermission = Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);
    if (!hasCreatePermission) return null;
    return <Selectable {...props} />;
  };
  WrappedSelectable.type = Selectable.type;
  return WrappedSelectable;
};

export const SELECTABLES: Base.Selectable[] = [
  ...EtherCAT.Task.SELECTABLES,
  ...HTTP.Task.SELECTABLES,
  ...LabJack.Task.SELECTABLES,
  ...Modbus.Task.SELECTABLES,
  ...NI.Task.SELECTABLES,
  ...OPC.Task.SELECTABLES,
  ...PagerDuty.Task.SELECTABLES,
  ...Slack.Task.SELECTABLES,
].map(withTaskVisibility);

export const Selector = Base.create(SELECTABLES, "Select a Task Type");
