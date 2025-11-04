// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Divider, Flex, Form } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type FC } from "react";

import { Device } from "@/hardware/ni/device";
import {
  CO_PULSE_OUTPUT_CHAN_TYPE,
  type COChannelType,
  type COIdleState,
} from "@/hardware/ni/task/types";

interface FormProps {
  path: string;
}

const IdleStateField = Form.buildSelectField<
  COIdleState,
  record.KeyedNamed<COIdleState>
>({
  fieldKey: "idleState",
  fieldProps: { label: "Idle State" },
  inputProps: {
    resourceName: "Idle State",
    data: [
      { key: "Low", name: "Low" },
      { key: "High", name: "High" },
    ],
  },
});

const UnitsField = Form.buildSelectField<string, record.KeyedNamed<string>>({
  fieldKey: "units",
  fieldProps: { label: "Scaled Units" },
  inputProps: {
    resourceName: "Scaled Units",
    data: [{ key: "Seconds", name: "Seconds" }],
    allowNone: false,
  },
});

const CHANNEL_FORMS: Record<COChannelType, FC<FormProps>> = {
  [CO_PULSE_OUTPUT_CHAN_TYPE]: ({ path }) => (
    <>
      <Flex.Box x>
        <Form.NumericField
          path={`${path}.initialDelay`}
          label="Initial Delay"
          inputProps={{ endContent: "s" }}
          grow
        />
        <Form.NumericField
          path={`${path}.highTime`}
          label="High Time"
          inputProps={{ endContent: "s" }}
          grow
        />
        <Form.NumericField
          path={`${path}.lowTime`}
          label="Low Time"
          inputProps={{ endContent: "s" }}
          grow
        />
      </Flex.Box>
      <Flex.Box x>
        <IdleStateField path={path} grow />
        <UnitsField path={path} grow />
      </Flex.Box>
    </>
  ),
};

export interface COChannelFormProps {
  type: COChannelType;
  path: string;
}

export const COChannelForm = ({ type, path }: COChannelFormProps) => {
  const FormComponent = CHANNEL_FORMS[type];
  return (
    <>
      <Flex.Box x wrap>
        <Device.PortField path={path} />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <FormComponent path={path} />
    </>
  );
};
