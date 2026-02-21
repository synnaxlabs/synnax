// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Form, Icon, Input } from "@synnaxlabs/pluto";
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

const Properties = () => (
  <>
    <Device.Select />
    <Flex.Box x grow>
      <Form.NumericField
        path="config.rate"
        label="Health Check Rate"
        inputProps={RATE_INPUT_PROPS}
      />
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const RATE_INPUT_PROPS = { endContent: "Hz" } as const;

type ScanFormProps = Common.Task.FormProps<typeof scanTypeZ, typeof scanConfigZ>;

const ScanForm: FC<ScanFormProps> = () => {
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
    <Flex.Box y gap="large" style={{ padding: "2rem" }}>
      <Form.TextField
        path="config.path"
        label="Endpoint Path"
        inputProps={PATH_INPUT_PROPS}
      />
      <Flex.Box x align="center" gap="small">
        <Input.Label>Response Validation</Input.Label>
        <Input.Switch value={hasResponse} onChange={handleToggleResponse} />
      </Flex.Box>
      {hasResponse && (
        <Flex.Box x grow>
          <Form.TextField
            grow
            path="config.response.field"
            label="JSON Pointer"
            inputProps={FIELD_INPUT_PROPS}
          />
          <Form.TextField
            grow
            path="config.response.expectedValue"
            label="Expected Value"
            inputProps={EXPECTED_VALUE_INPUT_PROPS}
          />
        </Flex.Box>
      )}
    </Flex.Box>
  );
};

const PATH_INPUT_PROPS = { placeholder: "/health" } as const;

const FIELD_INPUT_PROPS = { placeholder: "/status" } as const;

const EXPECTED_VALUE_INPUT_PROPS = { placeholder: "ok" } as const;

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
  Properties,
  Form: ScanForm,
  schemas: SCAN_SCHEMAS,
  type: SCAN_TYPE,
  getInitialValues,
  onConfigure,
  growForm: false,
});
