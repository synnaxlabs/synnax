// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query } from "@synnaxlabs/client";
import { Panel as PPanel, Status, Task as Base, Text } from "@synnaxlabs/pluto";
import { cloneElement } from "react";

import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { LabJack } from "@/feature/labjack";
import { Modbus } from "@/feature/modbus";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";
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
  ...OPCUA.Task.FORMS,
  ...PagerDuty.Task.FORMS,
};

const Content: Panel.Content = () => {
  const { key } = PPanel.useTabResource();
  const { type } = Base.use({ key });
  const Form = FORMS[type];
  if (Form == null)
    return (
      <Status.Summary
        variant="error"
        message={`No editor for task type ${type}`}
        center
      />
    );
  return <Form taskKey={key} />;
};

const Name: Panel.TabName = ({ allowRename = true }) => {
  const tabKey = PPanel.useTabKey();
  const { key } = PPanel.useTabResource();
  Base.useEnsure({ key });
  const name = Base.useName({ key });
  const { data } = Base.useResult({ key });
  const { update: rename } = Base.useRename();
  return (
    <>
      {getIcon(data?.type ?? "")}
      <Text.MaybeEditable
        id={Panel.tabNameID(tabKey)}
        value={name}
        disabled={!allowRename}
        onChange={(name) => rename({ key, name })}
      />
    </>
  );
};

const Icon: Panel.TabIcon = (props) => {
  const { key } = PPanel.useTabResource();
  const { data } = Base.useResult({ key });
  return cloneElement(getIcon(data?.type ?? ""), props);
};

export const TAB: Panel.Tab = {
  Content,
  Name,
  Icon,
  restore: async ({ client, resource }) => {
    const corpse = query.requireCorpse(client.tasks.getCached(resource.key));
    // The status goes with the instance, not the row: a restored task has
    // nothing running, so core seeds it as never deployed.
    await client.tasks.create({ ...corpse.payload, status: undefined });
  },
  useTombstone: Panel.createTombstoneReader(Base),
};
