// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/modbus/device/Connect.css";

import { type device, type rack, TimeSpan } from "@synnaxlabs/client";
import {
  Button,
  Device as PDevice,
  Flex,
  type Flux,
  Form,
  Nav,
  Rack,
  Status,
  Task,
} from "@synnaxlabs/pluto";
import { status as xstatus } from "@synnaxlabs/x";
import { useCallback } from "react";

import { CSS } from "@/css";
import { type Device, SCHEMAS, ZERO_PROPERTIES } from "@/hardware/modbus/device/types";
import {
  SCAN_SCHEMAS,
  SCAN_TYPE,
  TEST_CONNECTION_COMMAND_TYPE,
} from "@/hardware/modbus/task/types";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export const CONNECT_LAYOUT_TYPE = "configureModbusServer";

export const CONNECT_LAYOUT: Layout.BaseState = {
  key: CONNECT_LAYOUT_TYPE,
  type: CONNECT_LAYOUT_TYPE,
  name: "Server.Connect",
  icon: "Logo.Modbus",
  location: "modal",
  window: { resizable: false, size: { height: 500, width: 600 }, navTop: true },
};

const useForm = PDevice.createForm(SCHEMAS);

const INITIAL_VALUES: Device = {
  key: "",
  name: "Modbus Server",
  make: "Modbus",
  model: "Modbus",
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const beforeValidate = ({
  get,
  set,
}: Flux.BeforeValidateArgs<
  PDevice.RetrieveQuery,
  typeof PDevice.formSchema,
  PDevice.FluxSubStore
>) => {
  const host = get<string>("properties.connection.host").value;
  const port = get<number>("properties.connection.port").value;
  set("location", `${host}:${port}`);
};

const beforeSave = async ({
  client,
  get,
  store,
  set,
}: Flux.FormBeforeSaveParams<
  PDevice.RetrieveQuery,
  typeof PDevice.formSchema,
  PDevice.FluxSubStore
>) => {
  const scanTask = await Task.retrieveSingle({
    client,
    store,
    query: { type: SCAN_TYPE, rack: get<rack.Key>("rack").value },
    schemas: SCAN_SCHEMAS,
  });
  const state = await scanTask.executeCommandSync({
    type: TEST_CONNECTION_COMMAND_TYPE,
    timeout: TimeSpan.seconds(10),
    args: { connection: get("properties.connection").value },
  });
  if (state.variant === "error") throw new Error(state.message);
  // Since we just scanned successfully, we create a default healthy status for the
  // device that can then be overwritten by the scanner if we lose connection.
  const devStatus: device.Status = xstatus.create<typeof device.statusDetailsZ>({
    message: "Server connected",
    variant: "success",
    details: {
      rack: get<rack.Key>("rack").value,
      device: get<device.Key>("key").value,
    },
  });
  set("status", devStatus, { notifyOnChange: false, markTouched: false });
  return true;
};

export const Connect: Layout.Renderer = ({ layoutKey, onClose }) => {
  const { form, save, status, variant } = useForm({
    query: { key: layoutKey === CONNECT_LAYOUT_TYPE ? "" : layoutKey },
    initialValues: INITIAL_VALUES,
    beforeValidate,
    beforeSave,
    afterSave: useCallback(() => onClose(), [onClose]),
  });

  return (
    <Flex.Box align="start" className={CSS.B("modbus-connect")} justify="center">
      <Flex.Box className={CSS.B("content")} grow size="small">
        <Form.Form<typeof PDevice.formSchema> {...form}>
          <Form.TextField inputProps={NAME_INPUT_PROPS} path="name" />
          <Form.Field<rack.Key> path="rack" label="Connect From Location" required>
            {({ value, onChange }) => (
              <Rack.SelectSingle value={value} onChange={onChange} allowNone={false} />
            )}
          </Form.Field>
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
              label="Swap Bytes"
            />
            <Form.SwitchField
              path="properties.connection.swapWords"
              label="Swap Words"
            />
          </Flex.Box>
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Nav.Bar.Start gap="small">
          {variant == "success" ? (
            <Triggers.SaveHelpText action="Connect" noBar />
          ) : (
            <Status.Summary variant={variant} message={status.description} />
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Button.Button
            status={xstatus.keepVariants(variant, "loading")}
            onClick={() => save()}
            variant="filled"
          >
            Connect
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "Modbus Server",
} as const;

const HOST_INPUT_PROPS = { autoFocus: true, placeholder: "localhost" } as const;

const PORT_INPUT_PROPS = { placeholder: "502" } as const;
