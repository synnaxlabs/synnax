// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.

import "@/hardware/modbus/device/Connect.css";

import { rack, TimeSpan, UnexpectedError } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Form,
  Input,
  Nav,
  Rack,
  Status,
  Synnax,
} from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { v4 as uuid } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { NULL_CLIENT_ERROR } from "@/errors";
import { Common } from "@/hardware/common";
import {
  type ConnectionConfig,
  connectionConfigZ,
  MAKE,
  type Properties,
  ZERO_CONNECTION_CONFIG,
  ZERO_PROPERTIES,
} from "@/hardware/modbus/device/types";
import {
  SCAN_TYPE,
  TEST_CONNECTION_COMMAND_TYPE,
  type TestConnectionCommandState,
  type TestConnectionCommandStateDetails,
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

const formSchema = z.object({
  name: Common.Device.nameZ,
  rack: rack.keyZ,
  connection: connectionConfigZ,
});

interface FormSchema extends z.infer<typeof formSchema> {}

interface InternalProps extends Pick<Layout.RendererProps, "layoutKey" | "onClose"> {
  initialValues: FormSchema;
  properties?: Properties;
}

const Internal = ({ initialValues, layoutKey, onClose, properties }: InternalProps) => {
  const client = Synnax.use();
  const [connectionState, setConnectionState] = useState<TestConnectionCommandState>();
  const handleError = Status.useErrorHandler();
  const methods = Form.use({ values: initialValues, schema: formSchema });

  const testConnectionMutation = useMutation({
    onError: (e) => handleError(e, "Failed to test connection"),
    mutationFn: async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      if (!methods.validate("connection")) throw new Error("Invalid configuration");

      const rack = await client.hardware.racks.retrieve(
        methods.get<rack.Key>("rack").value,
      );

      const scanTasks = await rack.retrieveTaskByType(SCAN_TYPE);
      if (scanTasks.length === 0)
        throw new UnexpectedError(`No scan task found for driver ${rack.name}`);

      const task = scanTasks[0];
      const state = await task.executeCommandSync<TestConnectionCommandStateDetails>(
        TEST_CONNECTION_COMMAND_TYPE,
        {
          connection: methods.get("connection").value,
        },
        TimeSpan.seconds(10),
      );

      setConnectionState(state);
    },
  });

  const connectMutation = useMutation({
    onError: (e) => handleError(e, "Failed to connect to Modbus Server"),
    mutationFn: async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      if (!methods.validate()) throw new Error("Invalid configuration");

      // Test connection before saving
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
        location: `${methods.get<string>("connection.host").value}:${methods.get<number>("connection.port").value}`,
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

  const isPending = testConnectionMutation.isPending || connectMutation.isPending;

  return (
    <Align.Space align="start" className={CSS.B("modbus-connect")} justify="center">
      <Align.Space className={CSS.B("content")} grow size="small">
        <Form.Form<typeof formSchema> {...methods}>
          <Form.TextField
            inputProps={{
              level: "h2",
              placeholder: "Modbus Server",
              variant: "natural",
            }}
            path="name"
          />
          <Form.Field<rack.Key> path="rack" label="Connect From Location" required>
            {(p) => <Rack.SelectSingle {...p} allowNone={false} />}
          </Form.Field>
          <Align.Space direction="x" justify="spaceBetween">
            <Form.Field<string> grow path="connection.host">
              {(p) => <Input.Text autoFocus placeholder="localhost" {...p} />}
            </Form.Field>
            <Form.Field<number> path="connection.port">
              {(p) => <Input.Numeric placeholder="502" {...p} />}
            </Form.Field>
          </Align.Space>
          <Align.Space direction="x" justify="spaceBetween">
            <Form.Field<boolean> path="connection.swapBytes" label="Swap Bytes">
              {(p) => <Input.Switch {...p} />}
            </Form.Field>
            <Form.Field<boolean> path="connection.swapWords" label="Swap Words">
              {(p) => <Input.Switch {...p} />}
            </Form.Field>
          </Align.Space>
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
  const { isPending, isError, data } = useQuery<[FormSchema, Properties]>({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null || layoutKey === CONNECT_LAYOUT_TYPE)
        return [
          { name: "Modbus Server", connection: { ...ZERO_CONNECTION_CONFIG }, rack: 0 },
          deep.copy(ZERO_PROPERTIES),
        ];
      const dev = await client.hardware.devices.retrieve<Properties>(layoutKey);
      return [
        { name: dev.name, rack: dev.rack, connection: dev.properties.connection },
        dev.properties,
      ];
    },
  });

  if (isPending || isError) return null;

  const [initialValues, properties] = data;
  return (
    <Internal
      initialValues={initialValues}
      layoutKey={layoutKey}
      onClose={onClose}
      properties={properties}
    />
  );
};
