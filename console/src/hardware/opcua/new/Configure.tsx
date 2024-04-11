import { useState, type ReactElement } from "react";

import { DataType, TimeSpan } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Form,
  Input,
  Nav,
  Status,
  Steps,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { v4 as uuidv4 } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { CreateChannels } from "@/hardware/opcua/new/CreateChannels";
import { type Layout } from "@/layout";

import "@/hardware/opcua/new/Configure.css";

export const channelZ = z.object({
  dataType: z.string(),
  name: z.string(),
  nodeId: z.number(),
});

const groupZ = z.object({
  key: z.string(),
  name: z.string(),
  channels: channelZ.array(),
});

type Group = z.infer<typeof groupZ>;

const configureZ = z.object({
  name: z.string(),
  connection: connectionConfigZ,
  groups: groupZ.array(),
  role: z.enum(["index", "data"]),
});

export const connectWindowLayout: Layout.LayoutState = {
  key: "connectOPCUAServer",
  windowKey: "connectOPCUAServer",
  type: "connectOPCUAServer",
  name: "Connect OPC UA Server",
  location: "window",
  window: {
    resizable: false,
    size: { height: 800, width: 1000 },
    navTop: true,
  },
};

const STEPS: Steps.Step[] = [
  {
    key: "connect",
    title: "Connect",
  },
  {
    key: "createChannels",
    title: "Create Channels",
  },
];

export const Configure = (): ReactElement => {
  const client = Synnax.use();
  const [step, setStep] = useState("connect");

  const methods = Form.use<typeof configureZ>({
    values: {
      name: "",
      connection: {
        endpoint: "opc.tcp://0.0.0.0:4840",
        username: "",
        password: "",
      },
      groups: [],
    },
  });

  const handleNextStep = useMutation({
    mutationKey: [step, client?.key],
    mutationFn: async () => {
      if (!methods.validate() || client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      if (step === "connect") {
        const task = await rack.retrieveTaskByName("OPCUA Scanner");
        const state = await task.executeCommandSync(
          "scan",
          { connection: methods.get({ path: "connection" }).value },
          TimeSpan.seconds(1),
        );
        methods.set({
          path: "groups",
          value: [
            {
              key: nanoid(),
              name: "Group 1",
              channels: [
                ...state.details.map((c) => ({ ...c, role: "data", key: nanoid() })),
                { key: nanoid(), name: "Time", dataType: "timestamp", role: "index" },
              ],
            },
          ],
        });
        setStep("createChannels");
      } else if (step === "createChannels") {
        console.log("HELLO");
        const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
        console.log("BLOC");
        const task = await rack.retrieveTaskByName("OPCUA Scanner");
        const state = await task.executeCommandSync(
          "scan",
          { connection: methods.get({ path: "connection" }).value },
          TimeSpan.seconds(1),
        );
        try {
          await client.hardware.devices.create({
            key: uuidv4(),
            name: methods.get<string>({ path: "name" }).value,
            model: "opcua",
            make: "opcua",
            rack: rack.key,
            location: methods.get<string>({ path: "connection.endpoint" }).value,
            properties: {
              connection: methods.get({ path: "connection" }).value,
              channels: state.details,
            },
          });
        } catch (e) {
          console.error(e);
        }
        console.log("Created device");
        const groups = methods.get<Group[]>({ path: "groups" }).value;
        for (const group of groups) {
          console.log("ABC");
          const idx = await client.channels.create({
            name: group.name,
            isIndex: true,
            dataType: DataType.TIMESTAMP,
          });
          await client.channels.create(
            group.channels.map((c) => ({
              name: c.name,
              dataType: new DataType(c.dataType),
              index: idx.key,
            })),
          );
        }
      }
    },
  });

  let content: ReactElement;
  if (step === "connect") {
    content = <Connect />;
  } else if (step === "createChannels") {
    content = <CreateChannels />;
  }

  return (
    <Align.Space className={CSS.B("configure")} align="stretch" grow empty>
      <Form.Form {...methods}>
        <Align.Space className={CSS.B("content")} grow>
          {content}
        </Align.Space>
        <Nav.Bar size={48} location="bottom">
          <Nav.Bar.Start>
            <Steps.Steps value={step} onChange={setStep} steps={STEPS} />
          </Nav.Bar.Start>
          <Nav.Bar.End>
            <Button.Button variant="outlined">Cancel</Button.Button>
            <Button.Button
              onClick={() => handleNextStep.mutate()}
              loading={handleNextStep.isPending}
            >
              Next Step
            </Button.Button>
          </Nav.Bar.End>
        </Nav.Bar>
      </Form.Form>
    </Align.Space>
  );
};

const Connect = (): ReactElement => {
  const client = Synnax.use();

  const form = Form.useContext();

  const testConnection = useMutation({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (!form.validate() || client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      const task = await rack.retrieveTaskByName("OPCUA Scanner");
      console.log(form.get({ path: "connection" }));
      const state = await task.executeCommandSync(
        "test_connection",
        { connection: form.get({ path: "connection" }).value },
        TimeSpan.seconds(1),
      );
      console.log(state);
      return state;
    },
  });

  return (
    <Align.Space
      direction="x"
      justify="center"
      className={CSS.B("connect")}
      align="start"
      empty
      grow
      size={10}
    >
      <Align.Space className={CSS.B("description")} direction="y">
        <Text.Text level="h1">Let's connect your OPCUA Server</Text.Text>
        <Text.Text level="p">
          To start off, we'll need to know the connection details for your OPCUA server.
        </Text.Text>
      </Align.Space>
      <Align.Space
        direction="y"
        grow
        align="stretch"
        className={CSS.B("form")}
        style={{ padding: "2rem" }}
      >
        <Form.Field<string> path="name">
          {(p) => <Input.Text placeholder="Name" autoFocus {...p} />}
        </Form.Field>
        <Form.Field<string> path="connection.endpoint">
          {(p) => (
            <Input.Text placeholder="opc.tcp://localhost:4840" autoFocus {...p} />
          )}
        </Form.Field>
        <Align.Space direction="x" grow>
          <Form.Field<string> path="connection.username" grow>
            {(p) => <Input.Text placeholder="admin" {...p} />}
          </Form.Field>
          <Form.Field<string> path="connection.password" grow>
            {(p) => <Input.Text placeholder="password" {...p} />}
          </Form.Field>
        </Align.Space>
        <Align.Space direction="x">
          <Align.Space direction="x" grow>
            {testConnection.isError && (
              <Status.Text variant="error">{testConnection.error.message}</Status.Text>
            )}
            {testConnection.isSuccess && (
              <Status.Text variant={testConnection.data?.variant}>
                {testConnection.data?.details?.message}
              </Status.Text>
            )}
          </Align.Space>
          <Button.Button
            variant="outlined"
            loading={testConnection.isPending}
            disabled={testConnection.isPending}
            onClick={() => testConnection.mutate()}
          >
            Test Connection
          </Button.Button>
          {/* {testConnection.} */}
        </Align.Space>
      </Align.Space>
    </Align.Space>
  );
};
