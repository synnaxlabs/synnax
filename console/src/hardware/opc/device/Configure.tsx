// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/device/Configure.css";

import { rack as clientRack, TimeSpan } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Divider,
  Form,
  Input,
  Nav,
  Status,
  Synnax,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";
import { v4 as uuid } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { FS } from "@/fs";
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
import { Layout } from "@/layout";

export const CONFIGURE_LAYOUT_TYPE = "configureOPCServer";

export const CONFIGURE_LAYOUT: Layout.BaseState = {
  key: CONFIGURE_LAYOUT_TYPE,
  type: CONFIGURE_LAYOUT_TYPE,
  name: "Server.Connect",
  icon: "Logo.OPC",
  location: "modal",
  window: { resizable: false, size: { height: 710, width: 800 }, navTop: true },
};

const formSchema = z.object({
  name: z.string().min(1, "Name is required"),
  connection: connectionConfigZ,
});
interface FormSchema extends z.infer<typeof formSchema> {}

export const Configure: Layout.Renderer = ({ layoutKey, onClose }): ReactElement => {
  const client = Synnax.use();
  const { isPending, isError, data, error } = useQuery<[FormSchema, Properties]>({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null || layoutKey === CONFIGURE_LAYOUT_TYPE)
        return [
          { name: "New OPC UA Server", connection: { ...ZERO_CONNECTION_CONFIG } },
          deep.copy(ZERO_PROPERTIES),
        ];
      const dev = await client.hardware.devices.retrieve<Properties>(layoutKey);
      dev.properties = migrateProperties(dev.properties);
      return [
        { name: dev.name, connection: dev.properties.connection },
        dev.properties,
      ];
    },
  });
  if (isPending)
    return (
      <Status.Text.Centered variant="loading" level="h2">
        Loading Configuration from Synnax Server
      </Status.Text.Centered>
    );
  if (isError)
    return (
      <Align.Space direction="y" grow align="center" justify="center">
        <Text.Text level="h2" color={Status.variantColors.error}>
          Failed to load configuration for server with key {layoutKey}
        </Text.Text>
        <Text.Text level="p" color={Status.variantColors.error}>
          {error.message}
        </Text.Text>
      </Align.Space>
    );
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

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

interface InternalProps extends Layout.RendererProps {
  initialValues: FormSchema;
  properties?: Properties;
}

const Internal = ({
  initialValues,
  layoutKey,
  onClose,
  properties,
}: InternalProps): ReactElement => {
  const client = Synnax.use();
  const [connectionState, setConnectionState] =
    useState<TestConnectionCommandState | null>(null);
  const handleException = Status.useExceptionHandler();
  const methods = Form.use({ values: initialValues, schema: formSchema });
  const testConnectionMutation = useMutation({
    onError: (e) => handleException(e, "Failed to test connection"),
    mutationFn: async () => {
      if (client == null) throw new Error("Cannot reach Synnax server");
      if (!methods.validate("connection")) throw new Error("Invalid configuration");
      const rack = await client.hardware.racks.retrieve(
        clientRack.DEFAULT_CHANNEL_NAME,
      );
      const task = await rack.retrieveTaskByName(SCAN_NAME);
      const t = await task.executeCommandSync<TestConnectionCommandResponse>(
        TEST_CONNECTION_COMMAND,
        { connection: methods.get("connection").value },
        TimeSpan.seconds(10),
      );
      setConnectionState(t);
    },
  });
  const configureMutation = useMutation({
    onError: (e) => handleException(e, "Failed to connect to OPC UA Server"),
    mutationFn: async () => {
      if (client == null) throw new Error("Cannot reach Synnax server");
      if (!methods.validate()) throw new Error("Invalid configuration");
      await testConnectionMutation.mutateAsync();
      if (connectionState?.variant !== "success")
        throw new Error("Connection test failed");
      const rack = await client.hardware.racks.retrieve(
        clientRack.DEFAULT_CHANNEL_NAME,
      );
      const key = layoutKey === CONFIGURE_LAYOUT_TYPE ? uuid() : layoutKey;
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
  const hasSecurityPolicy =
    Form.useFieldValue<SecurityPolicy>("connection.securityMode", undefined, methods) !=
    NO_SECURITY_MODE;
  const isPending = testConnectionMutation.isPending || configureMutation.isPending;
  return (
    <Align.Space
      direction="y"
      justify="center"
      className={CSS.B("opc-configure")}
      align="start"
    >
      <Align.Space direction="y" grow size="small" className={CSS.B("content")}>
        <Form.Form {...methods}>
          <Form.TextField
            path="name"
            inputProps={{
              level: "h2",
              variant: "natural",
              placeholder: "OPC UA Server",
            }}
          />
          <Form.Field<string> path="connection.endpoint">
            {(p) => (
              <Input.Text placeholder="opc.tcp://localhost:4840" autoFocus {...p} />
            )}
          </Form.Field>
          <Divider.Divider direction="x" padded="bottom" />
          <Align.Space direction="x" justify="spaceBetween">
            <Form.Field<string> path="connection.username" grow>
              {(p) => <Input.Text placeholder="admin" {...p} />}
            </Form.Field>
            <Form.Field<string> path="connection.password" grow>
              {(p) => <Input.Text placeholder="password" type="password" {...p} />}
            </Form.Field>
            <Form.Field<SecurityMode>
              path="connection.securityMode"
              label="Security Mode"
            >
              {(p) => <SelectSecurityMode {...p} />}
            </Form.Field>
          </Align.Space>
          <Divider.Divider direction="x" padded="bottom" />
          <Form.Field<SecurityPolicy>
            path="connection.securityPolicy"
            label="Security Policy"
            grow={!hasSecurityPolicy}
          >
            {(p) => <SelectSecurityPolicy size="medium" {...p} />}
          </Form.Field>
          {hasSecurityPolicy && (
            <>
              <Form.Field<string>
                path="connection.clientCertificate"
                label="Client Certificate"
              >
                {(p) => <FS.InputFilePath grow {...p} />}
              </Form.Field>
              <Form.Field<string>
                path="connection.clientPrivateKey"
                label="Client Private Key"
              >
                {(p) => <FS.InputFilePath grow {...p} />}
              </Form.Field>
              <Form.Field<string>
                path="connection.serverCertificate"
                label="Server Certificate"
                grow
              >
                {(p) => <FS.InputFilePath grow {...p} />}
              </Form.Field>
            </>
          )}
        </Form.Form>
      </Align.Space>
      <Layout.BottomNavBar>
        <Nav.Bar.Start size="small">
          {connectionState == null ? (
            <>
              <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
              <Text.Text shade={7} level="small">
                To Test Connection
              </Text.Text>
            </>
          ) : (
            <Status.Text variant={connectionState.variant as Status.Variant} level="p">
              {connectionState.details?.message}
            </Status.Text>
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Button.Button
            variant="outlined"
            triggers={[SAVE_TRIGGER]}
            loading={testConnectionMutation.isPending}
            disabled={isPending}
            onClick={() => testConnectionMutation.mutate()}
          >
            Test Connection
          </Button.Button>
          <Button.Button
            disabled={isPending}
            loading={configureMutation.isPending}
            onClick={() => configureMutation.mutate()}
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Layout.BottomNavBar>
    </Align.Space>
  );
};
