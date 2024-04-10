import { type ReactElement } from "react";

import { Align, Button, Form, Input, Nav, Synnax } from "@synnaxlabs/pluto";
import { Series } from "@synnaxlabs/x";
import { z } from "zod";

import { CSS } from "@/css";
import { type Layout } from "@/layout";

const connectionConfigZ = z.object({
  endpoint: z.string(),
  username: z.string().optional(),
  password: z.string().optional(),
});

export const connectWindowLayout: Layout.LayoutState = {
  key: "connectOPCUAServer",
  windowKey: "connectOPCUAServer",
  type: "connectOPCUAServer",
  name: "Connect OPC UA Server",
  location: "window",
  window: {
    resizable: false,
    size: { height: 430, width: 650 },
    navTop: true,
  },
};

export const Connect = (): ReactElement => {
  const client = Synnax.use();

  const methods = Form.use<typeof connectionConfigZ>({
    values: {
      endpoint: "opc.tcp://0.0.0.0:4840",
      username: "",
      password: "",
    },
  });

  const handleTestConnection = async (): Promise<void> => {
    if (!methods.validate() || client == null) return;
    const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
    const task = await rack.retrieveTaskByName("OPCUA Scanner");
    const streamer = await client.telem.openStreamer({ channels: ["sy_task_state"] });
    const writer = await client.telem.openWriter({ channels: ["sy_task_cmd"] });
    const s = new Series([
      {
        task: task.key,
        type: "scan",
        args: {
          connection: methods.value(),
        },
      },
    ]);
    await writer.write("sy_task_cmd", s);
    for await (const frame of streamer) console.log(frame.at(-1));
  };

  return (
    <Align.Space className={CSS.B("configure")} align="stretch" empty>
      <Form.Form {...methods}>
        <Form.Field<string> path="endpoint">
          {(p) => (
            <Input.Text placeholder="opc.tcp://localhost:4840" autoFocus {...p} />
          )}
        </Form.Field>
        <Align.Space direction="x">
          <Form.Field<string> path="username">
            {(p) => <Input.Text placeholder="admin" {...p} />}
          </Form.Field>
          <Form.Field<string> path="password">
            {(p) => <Input.Text placeholder="password" {...p} />}
          </Form.Field>
        </Align.Space>
      </Form.Form>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.End className={CSS.BE("footer", "end")}>
          <Button.Button variant="text" onClick={handleTestConnection}>
            Test Connection
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};
