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
  Icon,
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

export const useRegisterModal = Modals.create(({ close }) => {
  const client = Synnax.use();
  const { form, save, variant } = User.useForm({
    query: {},
    afterSave: useCallback(() => close(), [close]),
  });

  return (
    <Modals.Frame>
      <Modals.Header icon={<Icon.User />}>User.Register</Modals.Header>
      <Modals.Body>
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
      </Modals.Body>
      <Modals.Footer>
        <Triggers.SaveHelpText action="Register" />
        <Nav.Bar.End>
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
      </Modals.Footer>
    </Modals.Frame>
  );
});
