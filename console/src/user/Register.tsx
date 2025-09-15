// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";
import { Button, Flex, Form, Nav, Status, Synnax } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";

import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

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

export const REGISTER_LAYOUT: Layout.BaseState = {
  key: REGISTER_LAYOUT_TYPE,
  type: REGISTER_LAYOUT_TYPE,
  icon: "User",
  location: "modal",
  name: "User.Register",
  window: {
    resizable: false,
    size: { height: 425, width: 650 },
    navTop: true,
  },
};

export const Register: Layout.Renderer = ({ onClose }) => {
  const client = Synnax.use();
  const methods = Form.use({ values: deep.copy(initialValues), schema: formSchema });
  const handleError = Status.useErrorHandler();

  const { mutate, isPending } = useMutation({
    mutationFn: async () => {
      if (!methods.validate()) return;
      const values = methods.value();
      if (client == null) throw new DisconnectedError();
      await client.users.create({ ...values });
      onClose();
    },
    onError: (e) => handleError(e, "Failed to register user"),
  });

  return (
    <Flex.Box grow empty>
      <Flex.Box
        className="console-form"
        justify="center"
        style={{ padding: "1rem 3rem" }}
        grow
      >
        <Form.Form<typeof formSchema> {...methods}>
          <Flex.Box y>
            <Flex.Box x grow>
              <Form.TextField
                path="firstName"
                label="First Name"
                inputProps={{
                  variant: "text",
                  level: "h2",
                  autoFocus: true,
                  placeholder: "Richard",
                  full: "x",
                }}
              />
              <Form.TextField
                path="lastName"
                label="Last Name"
                inputProps={{
                  variant: "text",
                  level: "h2",
                  placeholder: "Feynman",
                  full: "x",
                }}
              />
            </Flex.Box>
            <Form.TextField
              path="username"
              label="Username"
              inputProps={{
                placeholder: "username",
                full: "x",
              }}
            />
            <Form.TextField
              path="password"
              label="Password"
              inputProps={{
                placeholder: "password",
                type: "password",
                full: "x",
              }}
            />
          </Flex.Box>
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText action="Register" />
        <Nav.Bar.End style={{ paddingRight: "2rem" }}>
          <Button.Button
            onClick={() => mutate()}
            status={isPending ? "loading" : undefined}
            disabled={client == null}
            tooltip={
              client == null
                ? "No Cluster Connected"
                : `Save to ${client.props.name ?? "Synnax"}`
            }
            tooltipLocation="bottom"
            trigger={Triggers.SAVE}
            variant="filled"
          >
            Register
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
