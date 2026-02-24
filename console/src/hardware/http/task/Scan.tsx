// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Form, Icon, Input, JSON } from "@synnaxlabs/pluto";
import { type FC, useCallback } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/http/device";
import {
  type ResponseValidation,
  SCAN_SCHEMAS,
  SCAN_TYPE,
  type scanConfigZ,
  type scanTypeZ,
  ZERO_RESPONSE_VALIDATION,
  ZERO_SCAN_PAYLOAD,
} from "@/hardware/http/task/types";
import { Selector } from "@/selector";

export const SCAN_LAYOUT = {
  ...Common.Task.LAYOUT,
  type: SCAN_TYPE,
  name: ZERO_SCAN_PAYLOAD.name,
  icon: "Logo.HTTP",
} as const satisfies Common.Task.Layout;

export const ScanSelectable = Selector.createSimpleItem({
  title: "HTTP Scan Task",
  icon: <Icon.Logo.HTTP />,
  layout: SCAN_LAYOUT,
});

const RATE_INPUT_PROPS = { endContent: "Hz", style: { maxWidth: "20rem" } } as const;

const ExpectedValueField: FC = () => {
  const { set } = Form.useContext();
  const value = Form.useFieldValue<JSON.Primitive>("config.response.expectedValue");
  const valueType = JSON.detectType(value);

  const handleTypeChange = useCallback(
    (newType: JSON.PrimitiveTypeName) => {
      set("config.response.expectedValue", JSON.ZERO_VALUES[newType]);
    },
    [set],
  );

  return (
    <Flex.Box x align="start" gap="large">
      <Input.Item label="Response type">
        <JSON.SelectType value={valueType} onChange={handleTypeChange} />
      </Input.Item>
      {valueType === "string" && (
        <Form.TextField
          grow
          path="config.response.expectedValue"
          label="Expected response value"
          inputProps={EXPECTED_VALUE_INPUT_PROPS}
        />
      )}
      {valueType === "number" && (
        <Form.NumericField
          grow
          path="config.response.expectedValue"
          label="Expected response value"
          inputProps={EXPECTED_VALUE_INPUT_PROPS}
        />
      )}
      {valueType === "boolean" && (
        <Form.SwitchField
          path="config.response.expectedValue"
          label="Expected response value"
        />
      )}
    </Flex.Box>
  );
};

const ScanForm = () => {
  const { set } = Form.useContext();
  const hasResponse =
    Form.useFieldValue<ResponseValidation>("config.response", { optional: true }) !=
    null;
  const handleToggleResponse = useCallback(
    (enabled: boolean) => {
      set("config.response", enabled ? ZERO_RESPONSE_VALIDATION : undefined);
    },
    [set],
  );
  return (
    <Flex.Box y grow>
      <Flex.Box x align="start">
        <Device.Select />
        <Form.NumericField
          path="config.rate"
          label="Rate"
          inputProps={RATE_INPUT_PROPS}
        />
        <Common.Task.Fields.AutoStart />
      </Flex.Box>
      <Flex.Box x align="start" gap="large">
        <Form.TextField
          path="config.path"
          label="Health endpoint path"
          inputProps={PATH_INPUT_PROPS}
        />
        <Input.Item label="Validate response" align="start">
          <Input.Switch value={hasResponse} onChange={handleToggleResponse} />
        </Input.Item>
      </Flex.Box>
      {hasResponse && (
        <>
          <Form.TextField
            path="config.response.field"
            label="Response field (JSON Pointer)"
            inputProps={FIELD_INPUT_PROPS}
          />
          <ExpectedValueField />
        </>
      )}
    </Flex.Box>
  );
};

const PATH_INPUT_PROPS = {
  placeholder: "/health",
  style: { maxWidth: "20rem" },
} as const;

const FIELD_INPUT_PROPS = {
  placeholder: "/status",
  style: { maxWidth: "20rem" },
} as const;

const EXPECTED_VALUE_INPUT_PROPS = { style: { maxWidth: "20rem" } } as const;

const getInitialValues: Common.Task.GetInitialValues<
  typeof scanTypeZ,
  typeof scanConfigZ
> = ({ deviceKey }) => ({
  ...ZERO_SCAN_PAYLOAD,
  config: {
    ...ZERO_SCAN_PAYLOAD.config,
    device: deviceKey ?? ZERO_SCAN_PAYLOAD.config.device,
  },
});

const onConfigure: Common.Task.OnConfigure<typeof scanConfigZ> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  return [config, dev.rack];
};

export const Scan = Common.Task.wrapForm({
  Properties: ScanForm,
  schemas: SCAN_SCHEMAS,
  type: SCAN_TYPE,
  getInitialValues,
  onConfigure,
});
