// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access, status } from "@synnaxlabs/client";
import {
  Access,
  Button,
  Flex,
  Form,
  type Input,
  Nav,
  Synnax,
  User,
} from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Modals } from "@/layered/service/modals";
import { Triggers } from "@/triggers";

const FIRST_NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  variant: "text",
  level: "h2",
  autoFocus: true,
  placeholder: "Richard",
  full: "x",
};

const LAST_NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  variant: "text",
  level: "h2",
  placeholder: "Feynman",
  full: "x",
};

const USERNAME_INPUT_PROPS: Partial<Input.TextProps> = {
  placeholder: "username",
  full: "x",
};

const PASSWORD_INPUT_PROPS: Partial<Input.TextProps> = {
  placeholder: "password",
  type: "password",
  full: "x",
};

export const useOpenRegister = Modals.create<void>(
  { size: { height: 425, width: 650 } },
  ({ close }) => {
    const client = Synnax.use();
    const { form, save, variant } = User.useForm({
      query: {},
      afterSave: useCallback(() => close(), [close]),
    });

    return (
      <Flex.Box grow empty>
        <Modals.Header name="User.Register" icon="User" />
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
                  label="First"
                  inputProps={FIRST_NAME_INPUT_PROPS}
                />
                <Form.TextField
                  path="lastName"
                  label="Last"
                  inputProps={LAST_NAME_INPUT_PROPS}
                />
              </Flex.Box>
              <Form.TextField
                path="username"
                label="Username"
                inputProps={USERNAME_INPUT_PROPS}
              />
              <Form.TextField
                path="password"
                label="Password"
                inputProps={PASSWORD_INPUT_PROPS}
              />
              <Form.Field<access.role.Key> path="role" label="Role">
                {({ value, onChange }) => (
                  <Access.Role.Select value={value} onChange={onChange} />
                )}
              </Form.Field>
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
              trigger={Triggers.SAVE}
              variant="filled"
            >
              Register
            </Button.Button>
          </Nav.Bar.End>
        </Modals.BottomNavBar>
      </Flex.Box>
    );
  },
);
