// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Form, Nav, Synnax, User } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useCallback } from "react";

import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

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
  const { form, save, variant } = User.useForm({
    query: {},
    afterSave: useCallback(() => onClose(), [onClose]),
  });

  return (
    <Flex.Box grow empty>
      <Flex.Box
        className="console-form"
        justify="center"
        style={{ padding: "1rem 3rem" }}
        grow
      >
        <Form.Form<typeof User.formSchema> {...form}>
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
            onClick={() => save()}
            status={status.keepVariants(variant, "loading")}
            disabled={client == null}
            tooltip={
              client == null
                ? "No Core Connected"
                : `Save to ${client.params.name ?? "Synnax"}`
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
