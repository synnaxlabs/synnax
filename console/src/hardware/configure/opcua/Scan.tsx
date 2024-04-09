import { type ReactElement } from "react";

import { Align, Form, Input, Synnax } from "@synnaxlabs/pluto";
import { z } from "zod";

import { CSS } from "@/css";

const connectionConfigZ = z.object({
  endpoint: z.string(),
  username: z.string().optional(),
  password: z.string().optional(),
});

const ScanForm = (): ReactElement => {
  const client = Synnax.use();

  const methods = Form.use<typeof connectionConfigZ>({
    values: {
      endpoint: "",
      username: "",
      password: "",
    },
  });

  const handleTestConnection = async (): void => {
    if (!methods.validate() || client == null) return;
    const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
    const task = await rack.re;
    const w = await client.telem.openWriter({
      channels: ["sy_task_cmd"],
    });
    w.write({
      sy_task_cmd: [{}],
    });
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
    </Align.Space>
  );
};
