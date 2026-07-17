// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status, type Synnax as Client } from "@synnaxlabs/client";
import {
  Button,
  Component,
  Flex,
  Form as PForm,
  Header,
  Icon,
  Status,
} from "@synnaxlabs/pluto";
import { type FC, useCallback } from "react";

import { Select as SelectDevice } from "@/feature/slack/device/Select";
import {
  ALERT_SCHEMAS,
  ALERT_TYPE,
  type AlertPayload,
  type AlertSchemas,
  ZERO_ALERT_PAYLOAD,
} from "@/feature/slack/task/types";
import { Empty } from "@/platform/empty";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

export const ALERT_LAYOUT: Task.Layout = {
  ...Task.LAYOUT,
  type: ALERT_TYPE,
  name: ZERO_ALERT_PAYLOAD.name,
  icon: "Logo.Slack",
};

export const AlertSelectable = Selector.createSimpleItem({
  title: "Slack Alert",
  icon: <Icon.Logo.Slack />,
  layout: ALERT_LAYOUT,
});

const Properties = () => (
  <Flex.Box x grow>
    <SelectDevice />
    <PForm.TextField
      path="config.channel"
      label="Channel"
      inputProps={CHANNEL_INPUT_PROPS}
      grow
    />
    <Task.Fields.AutoStart />
  </Flex.Box>
);

const CHANNEL_INPUT_PROPS = { placeholder: "#alerts" };

const selectStatusRenderProp = Component.renderProp(
  (p: Omit<Status.SelectProps, "variant">) => <Status.Select {...p} />,
);

const BODY_STYLE = { padding: "2rem", overflowY: "auto" } as const;

const Form: FC<Task.FormProps<AlertSchemas>> = () => {
  const statuses = PForm.useFieldValue<status.Key[]>("config.statuses");
  const { set } = PForm.useContext();

  const handleAdd = useCallback(
    () => set("config.statuses", [...statuses, ""]),
    [set, statuses],
  );

  const handleRemove = useCallback(
    (index: number) =>
      set(
        "config.statuses",
        statuses.filter((_, i) => i !== index),
      ),
    [set, statuses],
  );

  return (
    <Flex.Box y grow empty>
      <Header.Header>
        <Header.Title weight={500} color={10}>
          Statuses
        </Header.Title>
        <Header.Actions>
          <Button.Button
            onClick={handleAdd}
            variant="text"
            contrast={2}
            tooltip="Add status"
            sharp
          >
            <Icon.Add />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      {statuses.length === 0 ? (
        <Empty.Action
          message="No statuses."
          action="Add a status"
          onClick={handleAdd}
          grow
        />
      ) : (
        <Flex.Box y style={BODY_STYLE} gap="small">
          {statuses.map((_, index) => (
            <Flex.Box key={index} x align="center" gap="small">
              <PForm.Field<string>
                path={`config.statuses.${index}`}
                showLabel={false}
                grow
                required
              >
                {selectStatusRenderProp}
              </PForm.Field>
              <Button.Button
                variant="text"
                onClick={() => handleRemove(index)}
                tooltip="Remove status"
              >
                <Icon.Close />
              </Button.Button>
            </Flex.Box>
          ))}
        </Flex.Box>
      )}
    </Flex.Box>
  );
};

const getInitialValues: Task.GetInitialValues<AlertSchemas> = ({
  deviceKey,
  config,
}) => {
  const pld: AlertPayload = { ...ZERO_ALERT_PAYLOAD };
  if (config != null) {
    const parsed = ALERT_SCHEMAS.config.safeParse(config);
    if (parsed.success) pld.config = parsed.data;
  }
  if (deviceKey != null) pld.config = { ...pld.config, device: deviceKey };
  return pld;
};

const onConfigure: Task.OnConfigure<AlertSchemas["config"]> = async (
  client: Client,
  config,
) => {
  const dev = await client.devices.retrieve({ key: config.device });
  return [config, dev.rack];
};

export const Alert = Task.wrapForm({
  Properties,
  Form,
  schemas: ALERT_SCHEMAS,
  type: ALERT_TYPE,
  getInitialValues,
  onConfigure,
});
