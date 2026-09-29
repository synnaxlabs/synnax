// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { ARINC429 } from "@/feature/arinc429";
import { CAN } from "@/feature/can";
import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { LabJack } from "@/feature/labjack";
import { MIL1553 } from "@/feature/mil1553";
import { Modbus } from "@/feature/modbus";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
import { PagerDuty } from "@/feature/pagerduty";
import { Serial } from "@/feature/serial";
import { TCP } from "@/feature/tcp";
import { UDP } from "@/feature/udp";
import { FLAGS } from "@/flags";
import { Panel } from "@/platform/panel";
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
  ...OPCUA.Task.SELECTABLES,
  ...PagerDuty.Task.SELECTABLES,
  ...(FLAGS.can
    ? [
        ...CAN.Task.SELECTABLES,
        ...Serial.Task.SELECTABLES,
        ...TCP.Task.SELECTABLES,
        ...UDP.Task.SELECTABLES,
        ...ARINC429.Task.SELECTABLES,
        ...MIL1553.Task.SELECTABLES,
      ]
    : []),
].map(withTaskVisibility);

export const Selector = Base.create({
  selectables: SELECTABLES,
  icon: <Icon.Task />,
  tabTitle: "Create task",
  text: "Create task",
});

export const SELECTOR_TAB_TYPE = "taskSelector";

export const useOpenSelector = (): (() => void) => {
  const openTab = Panel.useOpenTab();
  return useCallback(
    () => openTab({ variant: "view", type: SELECTOR_TAB_TYPE, args: {} }),
    [openTab],
  );
};
