// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access, status, UnexpectedError, user } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Access, Synnax, User } from "@synnaxlabs/pluto";

import { Core } from "@/platform/core";
import { Credential } from "@/platform/credential";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  variant: "text",
  level: "h2",
  autoFocus: true,
  placeholder: "Richard Feynman",
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

const Register = ({ close }: Modals.ContentProps<Record<never, never>, void>) => {
  const client = Synnax.use();
  const { password: passwordAccepted } = Core.useAcceptedMethods(client?.params);
  const openCredentials = Credential.useModal();
  const { form, save, variant } = User.useForm({
    query: null,
    afterSave: ({ value }) => {
      const { key, name } = value();
      if (key == null) throw new UnexpectedError("User key is null");
      close();
      openCredentials({
        subject: user.ontologyID(key),
        title: ["Credentials", name],
      });
    },
  });

  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.User />}>User.Register</Modal.Header>
      <Modal.Body>
        <Form.Form<typeof User.formSchema> {...form}>
          <Flex.Box y>
            <Form.TextField path="name" label="Name" inputProps={NAME_INPUT_PROPS} />
            <Form.TextField
              path="username"
              label="Username"
              visible={passwordAccepted}
              inputProps={USERNAME_INPUT_PROPS}
            />
            <Form.TextField
              path="password"
              label="Password"
              visible={passwordAccepted}
              inputProps={PASSWORD_INPUT_PROPS}
            />
            <Form.Field<access.role.Key> path="role" label="Role">
              {({ value, onChange }) => (
                <Access.Role.Select value={value} onChange={onChange} />
              )}
            </Form.Field>
          </Flex.Box>
        </Form.Form>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End>
          <Button.Button
            onClick={() => save()}
            status={status.keepVariants(variant, "loading")}
            disabled={client == null}
            trigger={Triggers.SAVE}
            triggerIndicator
            variant="filled"
          >
            Register
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

export const useRegisterModal = Modals.create(Register);
