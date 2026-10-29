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
  BAUD_RATES,
  BYTE_ORDERS,
  type ByteOrder,
  byteOrderOf,
  type Connection,
  type Device,
  INTEGRATION,
  PARITIES,
  type Parity,
  SCHEMAS,
  STOP_BITS,
  swapsOf,
  type Transport,
  type WriteCodes,
  ZERO_CONNECTIONS,
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

// The base covers the command round trip. Each attempt can spend one timeout to connect
// and one to wait for the reply.
const testConnectionTimeout = (
  { timeoutMs, retries }: Connection,
  delayMs: number,
): TimeSpan =>
  TimeSpan.seconds(10).add(
    TimeSpan.milliseconds((retries + 1) * (2 * timeoutMs + delayMs)),
  );

const INITIAL_CONNECTIONS = {
  tcp: { ...ZERO_CONNECTIONS.tcp, port: 502 },
  rtu: ZERO_CONNECTIONS.rtu,
} as const satisfies Record<Transport, Connection>;

const INITIAL_VALUES: Device = {
  key: "",
  name: "Modbus server",
  make: "Modbus",
  model: "Modbus",
  location: "",
  properties: { ...ZERO_PROPERTIES, connection: INITIAL_CONNECTIONS.tcp },
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
  const connection = get<Connection>("properties.connection").value;
  set(
    "location",
    connection.transport === "rtu"
      ? connection.serialPort
      : `${connection.host}:${connection.port}`,
  );
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
  const connection = get<Connection>("properties.connection").value;
  const state = await scanTask.executeCommandSync({
    type: TEST_CONNECTION_COMMAND_TYPE,
    timeout: testConnectionTimeout(
      connection,
      get<number>("properties.requests.delayMs").value,
    ),
    args: { connection },
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
  const transport = Form.useFieldValue<
    Transport,
    Transport,
    PDevice.FormSchema<typeof SCHEMAS>
  >("properties.connection.transport", { ctx: form });
  const [tab, setTab] = useState("unit");

  // The transports share no address fields, so a switch starts the address over.
  const renderTransport = ({ value, onChange }: Input.Control<Transport>) => {
    const handleChange = (value: Transport) => {
      const { swapBytes, swapWords, timeoutMs, retries } = form.get<Connection>(
        "properties.connection",
      ).value;
      form.set("properties.connection", {
        ...INITIAL_CONNECTIONS[value],
        swapBytes,
        swapWords,
        timeoutMs,
        retries,
      });
      onChange(value);
    };
    return <SelectTransport value={value} onChange={handleChange} />;
  };

  return (
    <Modal.Frame className={CSS.B("modbus-connect")}>
      <Modal.Header icon={<Icon.Logo.Modbus />}>{HEADER}</Modal.Header>
      <Modal.Body gap="large">
        <Form.Form<PDevice.FormSchema<typeof SCHEMAS>> {...form}>
          <Form.TextField path="name" showLabel={false} inputProps={NAME_INPUT_PROPS} />
          <Form.Sections>
            <Form.Section title="Connection">
              <Form.Field<Transport>
                path="properties.connection.transport"
                label="Transport"
                padHelpText={false}
              >
                {renderTransport}
              </Form.Field>
              {transport === "tcp" ? (
                <>
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
                </>
              ) : (
                <>
                  <Form.TextField
                    path="properties.connection.serialPort"
                    label="Serial port"
                    padHelpText={false}
                    inputProps={SERIAL_PORT_INPUT_PROPS}
                  />
                  <BaudRateField path="properties.connection.baudRate" />
                  <ParityField path="properties.connection.parity" />
                  <StopBitsField path="properties.connection.stopBits" />
                </>
              )}
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
                    inputProps={UNIT_ID_INPUT_PROPS[transport]}
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

const HEADER = ["Device", "Connect", "Modbus server"];

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

const SERIAL_PORT_INPUT_PROPS = {
  autoFocus: true,
  placeholder: "/dev/ttyUSB0",
} as const;

const UNIT_ID_INPUT_PROPS = {
  tcp: { bounds: { lower: 0, upper: 255 }, style: { width: "23rem" } },
  rtu: { bounds: { lower: 1, upper: 247 }, style: { width: "23rem" } },
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

const SelectTransport = (props: Select.ButtonsProps<Transport>) => (
  <Select.Buttons<Transport> {...props}>
    <Select.Item<Transport> itemKey="tcp">TCP</Select.Item>
    <Select.Item<Transport> itemKey="rtu">Serial</Select.Item>
  </Select.Buttons>
);

const numberItems = (values: readonly number[]): ReactElement[] =>
  values.map((v) => (
    <Select.Item<number> key={v} itemKey={v}>
      {v.toString()}
    </Select.Item>
  ));

const BaudRateField = Form.buildSelectField<number>({
  fieldProps: { label: "Baud rate", padHelpText: false },
  inputProps: {
    resourceName: "baud rate",
    allowNone: false,
    children: numberItems(BAUD_RATES),
  },
});

const PARITY_NAMES: Record<Parity, string> = {
  none: "None",
  even: "Even",
  odd: "Odd",
};

const ParityField = Form.buildSelectField<Parity>({
  fieldProps: { label: "Parity", padHelpText: false },
  inputProps: {
    resourceName: "parity",
    allowNone: false,
    children: PARITIES.map((parity) => (
      <Select.Item<Parity> key={parity} itemKey={parity}>
        {PARITY_NAMES[parity]}
      </Select.Item>
    )),
  },
});

const StopBitsField = Form.buildSelectField<number>({
  fieldProps: { label: "Stop bits", padHelpText: false },
  inputProps: {
    resourceName: "stop bits",
    allowNone: false,
    children: numberItems(STOP_BITS),
  },
});

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
