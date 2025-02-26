// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/device/Connect.css";

import { rack, TimeSpan } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Divider,
  Form,
  Input,
  Nav,
  Rack,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { v4 as uuid } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { NULL_CLIENT_ERROR } from "@/errors";
import { FS } from "@/fs";
import { Common } from "@/hardware/common";
import { SelectSecurityMode } from "@/hardware/opc/device/SelectSecurityMode";
import { SelectSecurityPolicy } from "@/hardware/opc/device/SelectSecurityPolicy";
import {
  type ConnectionConfig,
  connectionConfigZ,
  MAKE,
  migrateProperties,
  NO_SECURITY_MODE,
  type Properties,
  type SecurityMode,
  type SecurityPolicy,
  ZERO_CONNECTION_CONFIG,
  ZERO_PROPERTIES,
} from "@/hardware/opc/device/types";
import {
  SCAN_NAME,
  TEST_CONNECTION_COMMAND,
  type TestConnectionCommandResponse,
  type TestConnectionCommandState,
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
  window: { resizable: false, size: { height: 710, width: 800 }, navTop: true },
};

const formSchema = z.object({
  name: Common.Device.nameZ,
  rack: rack.keyZ,
  connection: connectionConfigZ,
});
interface FormSchema extends z.infer<typeof formSchema> {}

interface InternalProps extends Layout.RendererProps {
  initialValues: FormSchema;
  properties?: Properties;
}

const Internal = ({ initialValues, layoutKey, onClose, properties }: InternalProps) => {
  const client = Synnax.use();
  const [connectionState, setConnectionState] = useState<TestConnectionCommandState>();
  const handleException = Status.useExceptionHandler();
  const methods = Form.use({ values: initialValues, schema: formSchema });
  const testConnectionMutation = useMutation({
    onError: (e) => handleException(e, "Failed to test connection"),
    mutationFn: async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      if (!methods.validate("connection")) throw new Error("Invalid configuration");
      const rack = await client.hardware.racks.retrieve(
        methods.get<rack.Key>("rack").value,
      );
      const task = await rack.retrieveTaskByName(SCAN_NAME);
      const state = await task.executeCommandSync<TestConnectionCommandResponse>(
        TEST_CONNECTION_COMMAND,
        { connection: methods.get("connection").value },
        TimeSpan.seconds(10),
      );
      setConnectionState(state);
    },
  });
  const connectMutation = useMutation({
    onError: (e) => handleException(e, "Failed to connect to OPC UA Server"),
    mutationFn: async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      if (!methods.validate()) throw new Error("Invalid configuration");
      await testConnectionMutation.mutateAsync();
      if (connectionState?.variant !== "success")
        throw new Error("Connection test failed");
      const rack = await client.hardware.racks.retrieve(
        methods.get<rack.Key>("rack").value,
      );
      const key = layoutKey === CONNECT_LAYOUT_TYPE ? uuid() : layoutKey;
      await client.hardware.devices.create<Properties>({
        key,
        name: methods.get<string>("name").value,
        model: MAKE,
        make: MAKE,
        rack: rack.key,
        location: methods.get<string>("connection.endpoint").value,
        properties: {
          ...ZERO_PROPERTIES,
          ...properties,
          connection: methods.get<ConnectionConfig>("connection").value,
        },
        configured: true,
      });
      onClose();
    },
  });
  const hasSecurity =
    Form.useFieldValue<SecurityMode>("connection.securityMode", undefined, methods) !=
    NO_SECURITY_MODE;
  const isPending = testConnectionMutation.isPending || connectMutation.isPending;
  return (
    <Align.Space align="start" className={CSS.B("opc-connect")} justify="center">
      <Align.Space className={CSS.B("content")} grow size="small">
        <Form.Form {...methods}>
          <Form.TextField
            inputProps={{
              level: "h2",
              placeholder: "OPC UA Server",
              variant: "natural",
            }}
            path="name"
          />
          <Form.Field<rack.Key> path="rack" label="Connect From Location" required>
            {(p) => <Rack.SelectSingle {...p}></Rack.SelectSingle>}
          </Form.Field>
          <Form.Field<string> path="connection.endpoint">
            {(p) => (
              <Input.Text autoFocus placeholder="opc.tcp://localhost:4840" {...p} />
            )}
          </Form.Field>
          <Divider.Divider direction="x" padded="bottom" />
          <Align.Space direction="x" justify="spaceBetween">
            <Form.Field<string> grow path="connection.username">
              {(p) => <Input.Text placeholder="admin" {...p} />}
            </Form.Field>
            <Form.Field<string> grow path="connection.password">
              {(p) => <Input.Text placeholder="password" type="password" {...p} />}
            </Form.Field>
            <Form.Field<SecurityMode>
              label="Security Mode"
              path="connection.securityMode"
            >
              {SelectSecurityMode}
            </Form.Field>
          </Align.Space>
          <Divider.Divider direction="x" padded="bottom" />
          <Form.Field<SecurityPolicy>
            grow={!hasSecurity}
            path="connection.securityPolicy"
            label="Security Policy"
          >
            {(p) => <SelectSecurityPolicy size="medium" {...p} />}
          </Form.Field>
          {hasSecurity && (
            <>
              <Form.Field<string>
                label="Client Certificate"
                path="connection.clientCertificate"
              >
                {FS.InputFilePath}
              </Form.Field>
              <Form.Field<string>
                label="Client Private Key"
                path="connection.clientPrivateKey"
              >
                {FS.InputFilePath}
              </Form.Field>
              <Form.Field<string>
                grow
                label="Server Certificate"
                path="connection.serverCertificate"
              >
                {FS.InputFilePath}
              </Form.Field>
            </>
          )}
        </Form.Form>
      </Align.Space>
      <Modals.BottomNavBar>
        <Nav.Bar.Start size="small">
          {connectionState == null ? (
            <Triggers.SaveHelpText action="Test Connection" noBar />
          ) : (
            <Status.Text level="p" variant={connectionState.variant as Status.Variant}>
              {connectionState.details?.message}
            </Status.Text>
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Button.Button
            variant="outlined"
            triggers={Triggers.SAVE}
            loading={testConnectionMutation.isPending}
            disabled={isPending}
            onClick={() => testConnectionMutation.mutate()}
          >
            Test Connection
          </Button.Button>
          <Button.Button
            disabled={isPending}
            loading={connectMutation.isPending}
            onClick={() => connectMutation.mutate()}
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Align.Space>
  );
};

export const Connect: Layout.Renderer = ({ layoutKey, onClose }) => {
  const client = Synnax.use();
  const { isPending, isError, data, error } = useQuery<[FormSchema, Properties]>({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null || layoutKey === CONNECT_LAYOUT_TYPE)
        return [
          { name: "OPC UA Server", connection: { ...ZERO_CONNECTION_CONFIG }, rack: 0 },
          deep.copy(ZERO_PROPERTIES),
        ];
      const dev = await client.hardware.devices.retrieve<Properties>(layoutKey);
      dev.properties = migrateProperties(dev.properties);
      return [
        { name: dev.name, rack: dev.rack, connection: dev.properties.connection },
        dev.properties,
      ];
    },
  });
  if (isPending)
    return (
      <Status.Text.Centered level="h4" variant="loading">
        Loading Configuration from Synnax Server
      </Status.Text.Centered>
    );
  if (isError) {
    const color = Status.variantColors.error;
    return (
      <Align.Center style={{ padding: "3rem" }}>
        <Text.Text level="h2" color={color}>
          Failed to load configuration for server with key {layoutKey}
        </Text.Text>
        <Text.Text level="p" color={color}>
          {error.message}
        </Text.Text>
      </Align.Center>
    );
  }
  const [initialValues, properties] = data;
  return (
    <Internal
      focused
      initialValues={initialValues}
      layoutKey={layoutKey}
      onClose={onClose}
      properties={properties}
      visible
    />
  );
};
