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
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Device as PDevice, type Flux } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import {
  BYTE_ORDERS,
  type ByteOrder,
  byteOrderOf,
  type Device,
  INTEGRATION,
  SCHEMAS,
  swapsOf,
  type WriteCodes,
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

const TEST_CONNECTION_TIMEOUT = TimeSpan.seconds(10);

const INITIAL_VALUES: Device = {
  key: "",
  name: "Modbus server",
  make: "Modbus",
  model: "Modbus",
  location: "",
  properties: {
    ...ZERO_PROPERTIES,
    connection: { ...ZERO_PROPERTIES.connection, port: 502 },
  },
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
    afterSave: ({ get }) => {
      if (deviceKey == null) capture("device_connected", { integration: "modbus" });
      close(get<device.Key>("key").value);
    },
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
  const [tab, setTab] = useState("unit");

  return (
    <Modal.Frame className={CSS.B("modbus-connect")}>
      <Modal.Header icon={<Icon.Logo.Modbus />}>{HEADER}</Modal.Header>
      <Modal.Body gap="large">
        <Form.Form<PDevice.FormSchema<typeof SCHEMAS>> {...form}>
          <Form.TextField path="name" showLabel={false} inputProps={NAME_INPUT_PROPS} />
          <Form.Sections>
            <Form.Section title="Connection">
              <Form.TextField
                path="properties.connection.host"
                label="Host"
                padHelpText={false}
                inputProps={HOST_INPUT_PROPS}
              />
              <Form.NumericField
                path="properties.connection.port"
                label="Port"
                padHelpText={false}
                inputProps={PORT_INPUT_PROPS}
              />
              <PlatformDevice.RackField integration={INTEGRATION} />
            </Form.Section>
          </Form.Sections>
          <Tabs.Frame
            value={tab}
            onChange={setTab}
            className={CSS.BE("modbus-connect", "tabs")}
            empty={false}
            gap="large"
          >
            <Tabs.Selector variant="chip">
              <Tabs.Tab itemKey="unit">Unit</Tabs.Tab>
              <Tabs.Tab itemKey="advanced">Advanced</Tabs.Tab>
            </Tabs.Selector>
            <Tabs.Content itemKey="unit">
              <Form.Sections>
                <Form.Section>
                  <Form.NumericField
                    path="properties.connection.unitId"
                    label="Unit ID"
                    padHelpText={false}
                    inputProps={UNIT_ID_INPUT_PROPS}
                  />
                  <ByteOrderField />
                </Form.Section>
              </Form.Sections>
            </Tabs.Content>
            <Tabs.Content itemKey="advanced">
              <Form.Sections>
                <Form.Section title="Timing">
                  <Form.NumericField
                    path="properties.connection.timeoutMs"
                    label="Timeout"
                    padHelpText={false}
                    inputProps={TIMEOUT_INPUT_PROPS}
                  />
                  <Form.NumericField
                    path="properties.connection.retries"
                    label="Retries"
                    padHelpText={false}
                    inputProps={RETRIES_INPUT_PROPS}
                  />
                  <Form.NumericField
                    path="properties.requests.delayMs"
                    label="Request delay"
                    padHelpText={false}
                    inputProps={DELAY_INPUT_PROPS}
                  />
                </Form.Section>
                <Form.Section title="Requests">
                  <Form.NumericField
                    path="properties.requests.maxRegisters"
                    label="Max registers"
                    padHelpText={false}
                    inputProps={MAX_REGISTERS_INPUT_PROPS}
                  />
                  <Form.NumericField
                    path="properties.requests.maxGap"
                    label="Max gap"
                    padHelpText={false}
                    inputProps={MAX_GAP_INPUT_PROPS}
                  />
                  <Form.Field<WriteCodes>
                    path="properties.requests.writeCodes"
                    label="Write codes"
                    padHelpText={false}
                  >
                    {(props) => <SelectWriteCodes {...props} />}
                  </Form.Field>
                </Form.Section>
              </Form.Sections>
            </Tabs.Content>
          </Tabs.Frame>
        </Form.Form>
      </Modal.Body>
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

export const useConnectModal = PlatformDevice.createConnectModal(Connect, INTEGRATION);

const HEADER = ["Device", "Connect", "Modbus TCP server"];

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "Modbus server",
  "aria-label": "Name",
} as const;

const HOST_INPUT_PROPS = { autoFocus: true, placeholder: "localhost" } as const;

const PORT_INPUT_PROPS = {
  bounds: { lower: 1, upper: 65535 },
  style: { width: "23rem" },
} as const;

const UNIT_ID_INPUT_PROPS = {
  bounds: { lower: 0, upper: 255 },
  style: { width: "23rem" },
} as const;

const TIMEOUT_INPUT_PROPS = {
  bounds: { lower: 1, upper: 60000 },
  endContent: "ms",
  style: { width: "23rem" },
} as const;

const RETRIES_INPUT_PROPS = {
  bounds: { lower: 0, upper: 10 },
  style: { width: "23rem" },
} as const;

const DELAY_INPUT_PROPS = {
  bounds: { lower: 0, upper: 5000 },
  endContent: "ms",
  style: { width: "23rem" },
} as const;

const MAX_REGISTERS_INPUT_PROPS = {
  bounds: { lower: 1, upper: 125 },
  style: { width: "23rem" },
} as const;

const MAX_GAP_INPUT_PROPS = {
  bounds: { lower: 0, upper: 65535 },
  style: { width: "23rem" },
} as const;

const SWAP_BYTES_PATH = "properties.connection.swapBytes";
const SWAP_WORDS_PATH = "properties.connection.swapWords";

const BYTE_ORDER_NAMES: Record<ByteOrder, string> = {
  ABCD: "Big-endian",
  BADC: "Byte swap",
  CDAB: "Word swap",
  DCBA: "Little-endian",
};

// One select over the two stored swap settings.
const ByteOrderField = (): ReactElement => {
  const ctx = Form.useContext();
  const swapBytes = Form.useFieldValue<boolean>(SWAP_BYTES_PATH);
  const swapWords = Form.useFieldValue<boolean>(SWAP_WORDS_PATH);
  const handleChange = (order: ByteOrder) => {
    const swaps = swapsOf(order);
    ctx.set(SWAP_BYTES_PATH, swaps.swapBytes);
    ctx.set(SWAP_WORDS_PATH, swaps.swapWords);
  };
  return (
    <Input.Item label="Byte order" padHelpText={false}>
      <Select.Simple<ByteOrder>
        value={byteOrderOf({ swapBytes, swapWords })}
        onChange={handleChange}
        allowNone={false}
        resourceName="byte order"
      >
        {BYTE_ORDERS.map((order) => (
          <Select.Item<ByteOrder> key={order} itemKey={order}>
            {`${BYTE_ORDER_NAMES[order]} (${order})`}
          </Select.Item>
        ))}
      </Select.Simple>
    </Input.Item>
  );
};

const SelectWriteCodes = (props: Select.ButtonsProps<WriteCodes>) => (
  <Select.Buttons<WriteCodes> {...props}>
    <Select.Item<WriteCodes>
      itemKey="multiple"
      tooltip="Writes each run of adjacent addresses with FC15 or FC16"
      tooltipLocation="top"
    >
      Multiple
    </Select.Item>
    <Select.Item<WriteCodes>
      itemKey="single"
      tooltip="Writes each coil with FC05 and each register with FC06"
      tooltipLocation="top"
    >
      Single
    </Select.Item>
  </Select.Buttons>
);
