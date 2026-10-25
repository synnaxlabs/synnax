// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/modbus/device/Connect.css";

import { type device, type rack, status, TimeSpan } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Device as PDevice, type Flux } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import {
  type Device,
  INTEGRATION,
  SCHEMAS,
  ZERO_PROPERTIES,
} from "@/feature/modbus/device/types";
import {
  SCAN_SCHEMAS,
  SCAN_TYPE,
  TEST_CONNECTION_COMMAND_TYPE,
} from "@/feature/modbus/task/types";
import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Modals } from "@/platform/modals";

const TEST_CONNECTION_TIMEOUT = TimeSpan.seconds(10);

const INITIAL_VALUES: Device = {
  key: "",
  name: "Modbus server",
  make: "Modbus",
  model: "Modbus",
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const useForm = PDevice.createForm(SCHEMAS, INITIAL_VALUES);

const beforeValidate = ({
  get,
  set,
}: Flux.BeforeValidateParams<
  PDevice.RetrieveQuery,
  PDevice.FormSchema<typeof SCHEMAS>
>) => {
  const host = get<string>("properties.connection.host").value;
  const port = get<number>("properties.connection.port").value;
  set("location", `${host}:${port}`);
};

const beforeSave = async ({
  client,
  get,
  set,
}: Flux.FormBeforeSaveParams<
  PDevice.RetrieveQuery,
  PDevice.FormSchema<typeof SCHEMAS>
>) => {
  const scanTask = await client.tasks.retrieve({
    type: SCAN_TYPE,
    rack: get<rack.Key>("rack").value,
    includeStatus: true,
    schemas: SCAN_SCHEMAS,
  });
  const state = await scanTask.executeCommandSync({
    type: TEST_CONNECTION_COMMAND_TYPE,
    timeout: TEST_CONNECTION_TIMEOUT,
    args: { connection: get("properties.connection").value },
  });
  if (state.variant === "error") throw status.toError(state);
  // Since we just scanned successfully, we create a default healthy status for the
  // device that can then be overwritten by the scanner if we lose connection.
  const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
    name: get<string>("name").value,
    message: "Device connected",
    variant: "success",
    details: {
      rack: get<rack.Key>("rack").value,
      device: get<device.Key>("key").value,
    },
  });
  set("status", devStatus, { notifyOnChange: false, markTouched: false });
  return true;
};

export const Connect = ({
  deviceKey,
  rack: defaultRack,
  close,
  onBack,
}: PlatformDevice.ConnectProps) => {
  const { capture } = Analytics.use();
  const {
    form,
    save,
    status: stat,
    variant,
  } = useForm({
    query: deviceKey == null ? null : { key: deviceKey },
    initialValues: { ...INITIAL_VALUES, rack: defaultRack ?? 0 },
    beforeValidate,
    beforeSave,
    afterSave: useCallback(
      ({ get }) => {
        if (deviceKey == null) capture("device_connected", { integration: "modbus" });
        close(get<device.Key>("key").value);
      },
      [capture, close, deviceKey],
    ),
  });
  const rackKey = Form.useFieldValue<
    rack.Key,
    rack.Key,
    PDevice.FormSchema<typeof SCHEMAS>
  >("rack", { ctx: form });
  const reason = PlatformDevice.useBlockedReason({
    integration: INTEGRATION,
    name: "Modbus",
    rack: rackKey,
  });

  return (
    <Modal.Frame className={CSS.B("modbus-connect")}>
      <Modal.Header icon={<Icon.Logo.Modbus />}>Server.Connect</Modal.Header>
      <Flex.Box className={CSS.B("content")} grow size="small">
        <Form.Form<PDevice.FormSchema<typeof SCHEMAS>> {...form}>
          <Form.TextField inputProps={NAME_INPUT_PROPS} path="name" />
          <PlatformDevice.RackField integration={INTEGRATION} />
          <Flex.Box x justify="between">
            <Form.TextField
              grow
              path="properties.connection.host"
              inputProps={HOST_INPUT_PROPS}
            />
            <Form.NumericField
              path="properties.connection.port"
              inputProps={PORT_INPUT_PROPS}
            />
          </Flex.Box>
          <Flex.Box x justify="start">
            <Form.SwitchField
              path="properties.connection.swapBytes"
              label="Swap bytes"
            />
            <Form.SwitchField
              path="properties.connection.swapWords"
              label="Swap words"
            />
          </Flex.Box>
        </Form.Form>
      </Flex.Box>
      <PlatformDevice.ConnectFooter
        reason={reason}
        variant={variant}
        message={stat.description}
        onConnect={() => save()}
        onBack={onBack}
      />
    </Modal.Frame>
  );
};

export const useConnectModal = Modals.createPrompt<
  device.Key,
  PlatformDevice.ConnectParams
>(Connect);

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "Modbus server",
} as const;

const HOST_INPUT_PROPS = { autoFocus: true, placeholder: "localhost" } as const;

const PORT_INPUT_PROPS = { placeholder: "502" } as const;
