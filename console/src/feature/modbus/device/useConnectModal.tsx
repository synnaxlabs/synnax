// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/modbus/device/Connect.css";

import { type device, status, TimeSpan } from "@synnaxlabs/client";
import { Description } from "@synnaxlabs/lyra/description";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Device as PDevice, type Flux } from "@synnaxlabs/pluto";
import { deep, Density } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { ByteOrderOption } from "@/feature/modbus/device/ByteOrder";
import { MENU_PROPS, TRIGGER_PROPS } from "@/feature/modbus/device/Option";
import {
  BAUD_RATES,
  BYTE_ORDERS,
  type ByteOrder,
  byteOrderOf,
  type Connection,
  type Device,
  INTEGRATION_KEY,
  PARITIES,
  type Parity,
  SCHEMAS,
  statsOf,
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

const path = deep.path<Device>();
const connectionPath = path.properties.connection;
const tcp = deep.variant(connectionPath, "transport", "tcp");
const serial = deep.variant(connectionPath, "transport", "rtu");

// Matches PROBE_WAIT in the Driver: how long a probe waits for its turn on a connection
// that other units' requests keep busy.
const PROBE_WAIT = TimeSpan.seconds(10);

// The base covers the command round trip. Each attempt can spend one timeout to connect
// and one to wait for the reply.
const testConnectionTimeout = (
  { timeoutMs, retries }: Connection,
  delayMs: number,
): TimeSpan =>
  TimeSpan.seconds(10)
    .add(PROBE_WAIT)
    .add(TimeSpan.milliseconds((retries + 1) * (2 * timeoutMs + delayMs)));

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
  const connection = get(connectionPath).value;
  if (connection.transport === "rtu") {
    set(path.location, connection.serialPort);
    return;
  }
  const { host, port } = connection;
  // Brackets keep the colons of an IPv6 host apart from the port's.
  const shown = host.includes(":") && !host.startsWith("[") ? `[${host}]` : host;
  set(path.location, `${shown}:${port}`);
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
    rack: get(path.rack).value,
    one: true,
    includeStatus: true,
    schemas: SCAN_SCHEMAS,
  });
  const connection = get(connectionPath).value;
  const requests = get(path.properties.requests).value;
  const state = await scanTask.executeCommandSync({
    type: TEST_CONNECTION_COMMAND_TYPE,
    timeout: testConnectionTimeout(connection, requests.delayMs),
    args: { connection, requests },
  });
  if (state.variant === "error") throw status.toError(state);
  // Since we just scanned successfully, we create a default healthy status for the
  // device that can then be overwritten by the scanner if we lose connection.
  const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
    name: get(path.name).value,
    message: "Device connected",
    variant: "success",
    details: {
      rack: get(path.rack).value,
      connected: true,
    },
  });
  set(path.status, devStatus, { notifyOnChange: false, markTouched: false });
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
      close(get(path.key).value);
    },
  });
  const rackKey = Form.useFieldValue(path.rack, { ctx: form });
  const reason = PlatformDevice.useBlockedReason({
    integration: INTEGRATION_KEY,
    name: "Modbus",
    rack: rackKey,
  });
  const transport = Form.useFieldValue(connectionPath.transport, {
    ctx: form,
  });

  // The transports share no address fields, so a switch starts the address over.
  const renderTransport = ({ value, onChange }: Input.Control<Transport>) => {
    const handleChange = (value: Transport) => {
      const { swapBytes, swapWords, timeoutMs, retries } =
        form.get(connectionPath).value;
      form.set(connectionPath, {
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
    <PlatformDevice.ConnectFrame className={CSS.B("modbus-connect")}>
      <Modal.Header icon={<Icon.Logo.Modbus />}>{HEADER}</Modal.Header>
      <Modal.Body gap="large">
        <Form.Form<PDevice.FormSchema<typeof SCHEMAS>> {...form}>
          <Form.TextField
            path={path.name}
            showLabel={false}
            padHelpText={false}
            inputProps={NAME_INPUT_PROPS}
          />
          <Form.Sections className={SECTIONS_CLASS}>
            <Form.Section title="Connection">
              <Form.Field
                path={connectionPath.transport}
                label="Transport"
                padHelpText={false}
              >
                {renderTransport}
              </Form.Field>
              {transport === "tcp" ? (
                <Flex.Box x>
                  <Form.TextField
                    path={tcp.host}
                    label="Host"
                    padHelpText={false}
                    inputProps={HOST_INPUT_PROPS}
                  />
                  <Form.NumericField
                    path={tcp.port}
                    label="Port"
                    padHelpText={false}
                    inputProps={PORT_INPUT_PROPS}
                  />
                </Flex.Box>
              ) : (
                <>
                  <Flex.Box x>
                    <Form.TextField
                      path={serial.serialPort}
                      label="Serial port"
                      padHelpText={false}
                      inputProps={SERIAL_PORT_INPUT_PROPS}
                    />
                    <BaudRateField path={serial.baudRate} />
                  </Flex.Box>
                  <Flex.Box x>
                    <ParityField path={serial.parity} />
                    <StopBitsField path={serial.stopBits} />
                  </Flex.Box>
                </>
              )}
              <Flex.Box x>
                <Form.NumericField
                  path={connectionPath.unitId}
                  label="Unit ID"
                  padHelpText={false}
                  inputProps={UNIT_ID_INPUT_PROPS[transport]}
                />
                <ByteOrderField />
              </Flex.Box>
              <PlatformDevice.RackField integration={INTEGRATION_KEY} />
            </Form.Section>
          </Form.Sections>
          <Form.Sections className={SECTIONS_CLASS}>
            <Form.Section title="Timing">
              <Flex.Box x>
                <Form.TimeSpanField
                  path={connectionPath.timeoutMs}
                  label="Timeout"
                  unit={TimeSpan.MILLISECOND}
                  bounds={TIMEOUT_BOUNDS}
                  padHelpText={false}
                />
                <Form.NumericField
                  path={connectionPath.retries}
                  label="Retries"
                  padHelpText={false}
                  inputProps={RETRIES_INPUT_PROPS}
                />
                <Form.TimeSpanField
                  path={path.properties.requests.delayMs}
                  label="Request delay"
                  unit={TimeSpan.MILLISECOND}
                  bounds={DELAY_BOUNDS}
                  padHelpText={false}
                />
              </Flex.Box>
            </Form.Section>
            <Form.Section title="Requests">
              <Flex.Box x>
                <Form.NumericField
                  path={path.properties.requests.maxRegisters}
                  label="Max registers"
                  padHelpText={false}
                  inputProps={MAX_REGISTERS_INPUT_PROPS}
                />
                <Form.NumericField
                  path={path.properties.requests.maxGap}
                  label="Max gap"
                  padHelpText={false}
                  inputProps={MAX_GAP_INPUT_PROPS}
                />
              </Flex.Box>
              <Flex.Box x>
                <Form.Field
                  path={path.properties.requests.writeCodes}
                  label="Write codes"
                  padHelpText={false}
                >
                  {(props) => <SelectWriteCodes {...props} />}
                </Form.Field>
                <Form.CheckboxField
                  path={path.properties.requests.maskWriteSupported}
                  label="Mask write (FC22)"
                  padHelpText={false}
                />
              </Flex.Box>
            </Form.Section>
            {deviceKey != null && (
              <Form.Section title="Diagnostics">
                <Diagnostics />
              </Form.Section>
            )}
          </Form.Sections>
        </Form.Form>
      </Modal.Body>
      <PlatformDevice.ConnectFooter
        reason={reason}
        variant={variant}
        message={stat.description}
        onConnect={() => save()}
        onBack={onBack}
      />
    </PlatformDevice.ConnectFrame>
  );
};

export const useConnectModal = PlatformDevice.createConnectModal(
  Connect,
  INTEGRATION_KEY,
);

const HEADER = ["Device", "Connect", "Modbus server"];

const SECTIONS_CLASS = CSS.BE("modbus-connect", "sections");

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "Modbus server",
  "aria-label": "Name",
} as const;

const HOST_INPUT_PROPS = { autoFocus: true, placeholder: "localhost" } as const;

const PORT_INPUT_PROPS = { bounds: { lower: 1, upper: 65535 } } as const;

const SERIAL_PORT_INPUT_PROPS = {
  autoFocus: true,
  placeholder: "/dev/ttyUSB0",
} as const;

const UNIT_ID_INPUT_PROPS = {
  tcp: { bounds: { lower: 0, upper: 255 } },
  rtu: { bounds: { lower: 1, upper: 247 } },
} as const;

const TIMEOUT_BOUNDS = { lower: TimeSpan.MILLISECOND, upper: TimeSpan.MINUTE };

const RETRIES_INPUT_PROPS = { bounds: { lower: 0, upper: 10 } } as const;

const DELAY_BOUNDS = { upper: TimeSpan.seconds(5) };

const MAX_REGISTERS_INPUT_PROPS = {
  bounds: { lower: 1, upper: 125 },
} as const;

const MAX_GAP_INPUT_PROPS = { bounds: { lower: 0, upper: 65535 } } as const;

const formatResponse = (span: TimeSpan): string => `${span.milliseconds.toFixed(1)} ms`;

interface StatProps {
  name: string;
  value: string;
}

const Stat = ({ name, value }: StatProps): ReactElement => (
  <Description.Item>
    <Description.Label>{name}</Description.Label>
    <Description.Value>{value}</Description.Value>
  </Description.Item>
);

// The request counters the Driver reports in the device status.
const Diagnostics = (): ReactElement => {
  const stats = statsOf(Form.useFieldValue(path.status, { optional: true }));
  if (stats == null)
    return (
      <Status.Summary variant="info" level="small" message="No requests counted yet" />
    );
  return (
    <Description.List level="small" justify="between">
      <Stat name="Requests" value={stats.requests.toString()} />
      <Stat name="Retries" value={stats.retries.toString()} />
      <Stat name="Timeouts" value={stats.timeouts.toString()} />
      <Stat name="Exceptions" value={stats.exceptions.toString()} />
      <Stat name="Last response" value={formatResponse(stats.lastResponse)} />
      <Stat name="Longest response" value={formatResponse(stats.maxResponse)} />
    </Description.List>
  );
};

const SWAP_BYTES_PATH = connectionPath.swapBytes;
const SWAP_WORDS_PATH = connectionPath.swapWords;

// One select over the two stored swap settings.
const ByteOrderField = (): ReactElement => {
  const ctx = Form.useContext();
  const swapBytes = Form.useFieldValue(SWAP_BYTES_PATH);
  const swapWords = Form.useFieldValue(SWAP_WORDS_PATH);
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
        triggerProps={TRIGGER_PROPS}
        dialogProps={MENU_PROPS}
      >
        {BYTE_ORDERS.map((order) => (
          <Select.Item<ByteOrder> key={order} itemKey={order}>
            <ByteOrderOption order={order} size={Density.BIT32.valueOf()} />
          </Select.Item>
        ))}
      </Select.Simple>
    </Input.Item>
  );
};

const SelectTransport = (props: Select.ButtonsProps<Transport>) => (
  <Select.Buttons<Transport> {...props}>
    <Select.Item<Transport> itemKey="tcp">
      <Icon.Network />
      TCP
    </Select.Item>
    <Select.Item<Transport> itemKey="rtu">
      <Icon.Serial />
      Serial
    </Select.Item>
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
