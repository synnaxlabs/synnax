// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/device/Connect.css";

import { type device, type rack, TimeSpan } from "@synnaxlabs/client";
import {
  Button,
  Device,
  Divider,
  Flex,
  type Flux,
  Form,
  Input,
  Nav,
  Rack,
  Status,
  Task,
} from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { FS } from "@/fs";
import { SelectSecurityMode } from "@/hardware/opc/device/SelectSecurityMode";
import { SelectSecurityPolicy } from "@/hardware/opc/device/SelectSecurityPolicy";
import {
  type Make,
  NO_SECURITY_MODE,
  type Properties,
  type SecurityMode,
  type SecurityPolicy,
  ZERO_PROPERTIES,
} from "@/hardware/opc/device/types";
import {
  SCAN_SCHEMAS,
  SCAN_TYPE,
  TEST_CONNECTION_COMMAND_TYPE,
} from "@/hardware/opc/task/types";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export const CONNECT_LAYOUT_TYPE = "configureOPCServer";

export const CONNECT_LAYOUT: Layout.BaseState = {
  key: CONNECT_LAYOUT_TYPE,
  type: CONNECT_LAYOUT_TYPE,
  name: "Server.Connect",
  icon: "Logo.OPC",
  location: "modal",
  window: { resizable: false, size: { height: 720, width: 915 }, navTop: true },
};

const useForm = Device.createForm<Properties, Make>();

const INITIAL_VALUES: device.Device<Properties, Make> = {
  key: "",
  name: "OPC UA Server",
  make: "opc",
  model: "opc",
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
};

const beforeValidate = ({
  get,
  set,
}: Flux.BeforeValidateArgs<
  Device.UseRetrieveArgs,
  typeof Device.formSchema,
  Device.FluxStore
>) => set("location", get("properties.connection.endpoint").value);

const beforeSave = async ({
  client,
  get,
  store,
}: Flux.BeforeSaveArgs<
  Device.UseRetrieveArgs,
  typeof Device.formSchema,
  Device.FluxStore
>) => {
  const scanTask = await Task.retrieveSingle({
    client,
    store,
    params: { type: SCAN_TYPE, rack: get<rack.Key>("rack").value },
    schemas: SCAN_SCHEMAS,
  });
  const state = await scanTask.executeCommandSync(
    TEST_CONNECTION_COMMAND_TYPE,
    TimeSpan.seconds(10),
    { connection: get("properties.connection").value },
  );
  if (state.variant === "error") throw new Error(state.message);
  return true;
};

export const Connect: Layout.Renderer = ({ layoutKey, onClose }) => {
  const {
    form,
    save,
    status: stat,
    variant,
  } = useForm({
    params: { key: layoutKey === CONNECT_LAYOUT_TYPE ? "" : layoutKey },
    initialValues: INITIAL_VALUES,
    beforeValidate,
    beforeSave,
    afterSave: onClose,
  });

  const hasSecurity =
    Form.useFieldValue<SecurityMode, SecurityMode, typeof Device.formSchema>(
      "properties.connection.securityMode",
      { ctx: form },
    ) != NO_SECURITY_MODE;
  return (
    <Flex.Box align="start" className={CSS.B("opc-connect")} justify="center">
      <Flex.Box className={CSS.B("content")} grow gap="small">
        <Form.Form<typeof Device.formSchema> {...form}>
          <Form.TextField
            inputProps={{
              level: "h2",
              placeholder: "OPC UA Server",
              variant: "text",
            }}
            path="name"
          />
          <Form.Field<rack.Key> path="rack" label="Connect From" required>
            {({ value, onChange }) => (
              <Rack.SelectSingle value={value} onChange={onChange} allowNone={false} />
            )}
          </Form.Field>
          <Form.Field<string> path="properties.connection.endpoint">
            {(p) => (
              <Input.Text autoFocus placeholder="opc.tcp://localhost:4840" {...p} />
            )}
          </Form.Field>
          <Divider.Divider x padded="bottom" />
          <Flex.Box x justify="between">
            <Form.Field<string> grow path="properties.connection.username">
              {(p) => <Input.Text placeholder="admin" {...p} />}
            </Form.Field>
            <Form.Field<string> grow path="properties.connection.password">
              {(p) => <Input.Text placeholder="password" type="password" {...p} />}
            </Form.Field>
            <Form.Field<SecurityMode>
              label="Security Mode"
              path="properties.connection.securityMode"
            >
              {({ value, onChange }) => (
                <SelectSecurityMode value={value} onChange={onChange} />
              )}
            </Form.Field>
          </Flex.Box>
          <Divider.Divider x padded="bottom" />
          <Form.Field<SecurityPolicy>
            grow={!hasSecurity}
            path="properties.connection.securityPolicy"
            label="Security Policy"
          >
            {({ value, onChange }) => (
              <SelectSecurityPolicy value={value} onChange={onChange} />
            )}
          </Form.Field>
          {hasSecurity && (
            <>
              <Form.Field<string>
                label="Client Certificate"
                path="properties.connection.clientCertificate"
              >
                {FS.InputFilePath}
              </Form.Field>
              <Form.Field<string>
                label="Client Private Key"
                path="properties.connection.clientPrivateKey"
              >
                {FS.InputFilePath}
              </Form.Field>
              <Form.Field<string>
                grow
                label="Server Certificate"
                path="properties.connection.serverCertificate"
              >
                {FS.InputFilePath}
              </Form.Field>
            </>
          )}
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Nav.Bar.Start gap="small">
          {variant == "success" ? (
            <Triggers.SaveHelpText action="Test Connection" noBar />
          ) : (
            <Status.Summary status={variant} message={stat.description} />
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Button.Button
            status={status.keepVariants(variant, "loading")}
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
