// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type rack, TimeSpan } from "@synnaxlabs/client";
import {
  Button,
  Device,
  Flex,
  type Flux,
  Form,
  Input,
  Nav,
  Rack,
  Status,
  Task,
  Text,
} from "@synnaxlabs/pluto";
import { status as xstatus } from "@synnaxlabs/x";
import { useCallback } from "react";

import { CSS } from "@/css";
import {
  type Make,
  type Model,
  type Properties,
  ZERO_PROPERTIES,
} from "@/hardware/visa/device/types";
import {
  SCAN_SCHEMAS,
  SCAN_TYPE,
  TEST_CONNECTION_COMMAND_TYPE,
} from "@/hardware/visa/task/types";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export const CONNECT_LAYOUT_TYPE = "configureVISADevice";

export const CONNECT_LAYOUT: Layout.BaseState = {
  key: CONNECT_LAYOUT_TYPE,
  type: CONNECT_LAYOUT_TYPE,
  name: "Device.Connect",
  icon: "Logo.VISA",
  location: "modal",
  window: { resizable: false, size: { height: 550, width: 650 }, navTop: true },
};

const useForm = Device.createForm<Properties, Make, Model>();

const INITIAL_VALUES: device.Device<Properties, Make, Model> = {
  key: "",
  name: "VISA Instrument",
  make: "Generic",
  model: "VISA",
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const beforeValidate = ({
  get,
  set,
}: Flux.BeforeValidateArgs<
  Device.RetrieveQuery,
  typeof Device.formSchema,
  Device.FluxSubStore
>) => {
  const resourceName = get<string>("properties.connection.resourceName").value;
  set("location", resourceName);
};

const beforeSave = async ({
  client,
  get,
  store,
}: Flux.FormBeforeSaveParams<
  Device.RetrieveQuery,
  typeof Device.formSchema,
  Device.FluxSubStore
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
    args: { resource_name: get("properties.connection.resourceName").value },
  });
  if (state.variant === "error") throw new Error(state.message);
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
    <Flex.Box align="start" className={CSS.B("visa-connect")} justify="center">
      <Flex.Box className={CSS.B("content")} grow size="small">
        <Form.Form<typeof Device.formSchema> {...form}>
          <Form.TextField
            inputProps={{
              level: "h2",
              placeholder: "VISA Instrument",
              variant: "text",
            }}
            path="name"
          />
          <Form.Field<rack.Key> path="rack" label="Connect From Location" required>
            {({ value, onChange }) => (
              <Rack.SelectSingle value={value} onChange={onChange} allowNone={false} />
            )}
          </Form.Field>
          <Form.Field<string>
            path="properties.connection.resourceName"
            label="Resource Name"
            required
          >
            {(p) => (
              <Input.Text
                autoFocus
                placeholder="e.g. TCPIP0::192.168.1.100::INSTR or USB0::0x1234::0x5678::INSTR"
                {...p}
              />
            )}
          </Form.Field>
          <Text.Text level="small" shade={6}>
            Examples: TCPIP0::192.168.1.100::INSTR (Ethernet), USB0::0x1234::0x5678::INSTR
            (USB), GPIB0::10::INSTR (GPIB), ASRL1::INSTR (Serial)
          </Text.Text>
          <Flex.Box x justify="between">
            <Form.Field<number>
              path="properties.connection.timeoutMs"
              label="Timeout (ms)"
            >
              {(p) => <Input.Numeric placeholder="5000" {...p} />}
            </Form.Field>
            <Form.Field<boolean>
              path="properties.connection.termCharEnabled"
              label="Term Char Enabled"
            >
              {(p) => <Input.Switch {...p} />}
            </Form.Field>
          </Flex.Box>
          <Form.Field<string>
            path="properties.connection.termChar"
            label="Termination Character"
            optional
          >
            {(p) => (
              <Input.Text
                placeholder="\n"
                disabled={!form.get("properties.connection.termCharEnabled").value}
                {...p}
              />
            )}
          </Form.Field>
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