// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Panel as PPanel, Status, Task as Base, Text } from "@synnaxlabs/pluto";

import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { NI } from "@/feature/ni";
import { OPC } from "@/feature/opc";
import { PagerDuty } from "@/feature/pagerduty";
import { getIcon } from "@/feature/task/types";
import { Panel } from "@/platform/panel";
import { type Task } from "@/platform/task";

export const FORMS: Task.Forms = {
  ...EtherCAT.Task.FORMS,
  ...HTTP.Task.FORMS,
  ...LabJack.Task.FORMS,
  ...Modbus.Task.FORMS,
  ...NI.Task.FORMS,
  ...OPC.Task.FORMS,
  ...PagerDuty.Task.FORMS,
};

const Content: Panel.Content = () => {
  const { key } = PPanel.useSelectTabResource();
  const { data, status, variant } = Base.useRetrieve({ key });
  if (variant !== "success" || data == null)
    return <Status.Summary status={status} center />;
  const Form = FORMS[data.type];
  if (Form == null)
    return (
      <Status.Summary
        variant="error"
        message={`No editor for task type ${data.type}`}
        center
      />
    );
  return <Form taskKey={key} />;
};

const Name: Panel.TabName = () => {
  const tabKey = PPanel.useTabKey();
  const { key } = PPanel.useSelectTabResource();
  const { data } = Base.useRetrieve({ key });
  const { update: rename } = Base.useRename();
  return (
    <>
      {getIcon(data?.type ?? "")}
      <Text.Editable
        id={Panel.tabNameID(tabKey)}
        value={data?.name ?? ""}
        onChange={(name) => rename({ key, name })}
      />
    </>
  );
};

export const TAB: Panel.Tab = { Content, Name };
