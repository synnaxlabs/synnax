// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Align,
  Button,
  Form,
  Input,
  Nav,
  Status,
  Synnax,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement } from "react";
import { z } from "zod";

import { Layout } from "@/layout";
import { Permissions } from "@/permissions";

const formSchema = z.object({
  username: z.string().min(1, "Username must not be empty"),
  password: z.string().min(1, "Password must not be empty"),
  firstName: z.string().min(1, "First Name must not be empty"),
  lastName: z.string().min(1, "Last Name must not be empty"),
});
type FormValues = z.infer<typeof formSchema>;

const initialValues: FormValues = {
  username: "",
  password: "",
  firstName: "",
  lastName: "",
};

export const REGISTER_LAYOUT_TYPE = "registerUser";

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const registerLayout = ({
  window,
  ...rest
}: Partial<Layout.State>): Layout.State => ({
  ...rest,
  key: REGISTER_LAYOUT_TYPE,
  type: REGISTER_LAYOUT_TYPE,
  windowKey: REGISTER_LAYOUT_TYPE,
  icon: "User",
  location: "modal",
  name: "User.Register",
  window: {
    resizable: false,
    size: { height: 370, width: 700 },
    navTop: true,
    ...window,
  },
});

export const RegisterModal = ({ onClose }: Layout.RendererProps): ReactElement => {
  const client = Synnax.use();
  const methods = Form.use({ values: deep.copy(initialValues), schema: formSchema });
  const addStatus = Status.useAggregator();

  const { mutate, isPending } = useMutation({
    mutationFn: async () => {
      if (!methods.validate()) return;
      const values = methods.value();
      if (client == null) throw new Error("No Cluster Connected");
      const user = await client.user.create({ ...values });
      Permissions.setBasePermissions(client, user.key);
      onClose();
    },
    onError: (e) =>
      addStatus({
        message: "Failed to register user",
        description: e.message,
        variant: "error",
      }),
  });

  return (
    <Align.Space style={{ paddingTop: "2rem", height: "100%" }} grow empty>
      <Align.Space
        className="console-form"
        justify="center"
        style={{ padding: "1rem 3rem" }}
        grow
      >
        <Form.Form {...methods}>
          <Align.Space direction="y">
            <Form.Field<string> path="username">
              {(p) => (
                <Input.Text
                  autoFocus
                  level="h2"
                  variant="natural"
                  placeholder="username"
                  {...p}
                />
              )}
            </Form.Field>
            <Form.Field<string> path="password">
              {(p) => (
                <Input.Text
                  level="h2"
                  variant="natural"
                  placeholder="password"
                  type="password"
                  {...p}
                />
              )}
            </Form.Field>
            <Align.Space direction="x">
              <Form.Field<string> path="firstName" label="First Name">
                {(p) => (
                  <Input.Text level="h3" variant="natural" placeholder="first" {...p} />
                )}
              </Form.Field>
              <Form.Field<string> path="lastName" label="Last Name">
                {(p) => (
                  <Input.Text level="h3" variant="natural" placeholder="last" {...p} />
                )}
              </Form.Field>
            </Align.Space>
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.Start style={{ paddingLeft: "2rem" }} size="small">
          <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
          <Text.Text shade={7} level="small">
            To Register
          </Text.Text>
        </Nav.Bar.Start>
        <Nav.Bar.End style={{ paddingRight: "2rem" }}>
          <Button.Button
            onClick={() => mutate()}
            disabled={client == null || isPending}
            tooltip={
              client == null
                ? "No Cluster Connected"
                : `Save to ${client.props.name ?? "Synnax"}`
            }
            tooltipLocation="bottom"
            loading={isPending}
            triggers={[SAVE_TRIGGER]}
          >
            Register in Synnax
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};
