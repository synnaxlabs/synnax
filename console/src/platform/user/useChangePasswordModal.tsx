// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, type user } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  Form,
  Icon,
  type Input,
  Nav,
  Synnax,
  User,
} from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement, useCallback } from "react";

import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const INPUT_PROPS: Partial<Input.TextProps> = { type: "password", full: "x" };

const FIRST_INPUT_PROPS: Partial<Input.TextProps> = {
  ...INPUT_PROPS,
  autoFocus: true,
};

interface FrameProps extends PropsWithChildren {
  title: string | string[];
  save: () => void;
  variant: status.Variant;
}

const Frame = ({ title, save, variant, children }: FrameProps): ReactElement => {
  const client = Synnax.use();
  return (
    <Modals.Frame>
      <Modals.Header icon={<Icon.Lock />}>{title}</Modals.Header>
      <Modals.Body>
        <Flex.Box y>{children}</Flex.Box>
      </Modals.Body>
      <Modals.Footer>
        <Triggers.SaveHelpText action="Change" />
        <Nav.Bar.End>
          <Button.Button
            onClick={save}
            variant="filled"
            disabled={client == null}
            status={status.keepVariants(variant, "loading")}
            tooltip={client == null ? "No Core connected" : undefined}
            tooltipLocation="bottom"
            trigger={Triggers.SAVE}
          >
            Change
          </Button.Button>
        </Nav.Bar.End>
      </Modals.Footer>
    </Modals.Frame>
  );
};

export interface ChangePasswordModalParams {
  /** Key of the user whose password is being set. */
  userKey: user.Key;
  title?: string | string[];
}

/** Sets another user's password. Requires update access on that user. */
export const useChangePasswordModal = Modals.create<ChangePasswordModalParams>(
  ({ title, close, userKey }) => {
    const { form, save, variant } = User.useChangePasswordForm({
      query: null,
      initialValues: { key: userKey, password: "", confirmPassword: "" },
      afterSave: useCallback(() => close(), [close]),
    });
    return (
      <Form.Form<typeof User.changePasswordFormSchema> {...form}>
        <Frame title={title ?? "Password.Change"} save={() => save()} variant={variant}>
          <Form.TextField
            path="password"
            label="New password"
            inputProps={FIRST_INPUT_PROPS}
          />
          <Form.TextField
            path="confirmPassword"
            label="Confirm password"
            inputProps={INPUT_PROPS}
          />
        </Frame>
      </Form.Form>
    );
  },
);

/**
 * Sets the signed-in user's own password. The current password is asked for so that a
 * session left unattended cannot be used to change it.
 */
export const useChangeOwnPasswordModal = Modals.create(({ close }) => {
  const { form, save, variant } = User.useChangeOwnPasswordForm({
    query: null,
    afterSave: useCallback(() => close(), [close]),
  });
  return (
    <Form.Form<typeof User.changeOwnPasswordFormSchema> {...form}>
      <Frame title="Password.Change" save={() => save()} variant={variant}>
        <Form.TextField
          path="currentPassword"
          label="Current password"
          inputProps={FIRST_INPUT_PROPS}
        />
        <Form.TextField path="password" label="New password" inputProps={INPUT_PROPS} />
        <Form.TextField
          path="confirmPassword"
          label="Confirm password"
          inputProps={INPUT_PROPS}
        />
      </Frame>
    </Form.Form>
  );
});
