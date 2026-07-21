// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  Form,
  type Input,
  Status,
  Synnax,
  Text,
  type Triggers,
} from "@synnaxlabs/pluto";
import { TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import { z } from "zod";

import { CSS } from "@/platform/css";
import { Session } from "@/session";

export const SIGN_IN_TRIGGER: Triggers.Trigger = ["Enter"];

export const credentialsZ = z.object({
  username: z.string().min(1, "Username is required"),
  password: z.string().min(1, "Password is required"),
});

export interface Credentials extends z.infer<typeof credentialsZ> {}

export const USERNAME_INPUT_PROPS: Partial<Input.TextProps> = {
  placeholder: "synnax",
  autoFocus: true,
  size: "large",
};

export const PASSWORD_INPUT_PROPS: Partial<Input.TextProps> = {
  placeholder: "seldon",
  type: "password",
  size: "large",
};

const REAUTH_TIMEOUT = TimeSpan.seconds(10);

export interface CredentialsFormProps {
  onSuccess?: () => void;
}

/**
 * Credential re-entry for an active cluster whose connection is in error(auth).
 * Submits through the client's connection machine and persists the working
 * credentials to the session on success.
 */
export const CredentialsForm = ({ onSuccess }: CredentialsFormProps): ReactElement => {
  const client = Synnax.use();
  const activeKey = Session.Cluster.useSelectSelectedKey();
  const cluster = Session.Cluster.useSelectState(activeKey ?? undefined);
  const dispatch = Session.useDispatch();
  const [stat, setStat] = useState<status.Status>(() =>
    status.create({ variant: "disabled", message: "" }),
  );
  const methods = Form.use<typeof credentialsZ>({
    schema: credentialsZ,
    values: { username: cluster?.username ?? "", password: "" },
  });

  const handleSubmit = (): void =>
    void (async () => {
      if (!methods.validate() || client == null || cluster == null) return;
      const credentials = methods.value();
      setStat(status.create({ variant: "loading", message: "Signing in..." }));
      client.reauthenticate(credentials);
      try {
        await client.connect({ timeout: REAUTH_TIMEOUT });
        dispatch(Session.Cluster.set({ ...cluster, ...credentials }));
        onSuccess?.();
      } catch (error) {
        setStat(status.fromException(error, "Failed to sign in"));
      }
    })();

  return (
    <Form.Form<typeof credentialsZ> {...methods}>
      <Flex.Box y align="center" grow gap="huge" shrink={false}>
        <Text.Text level="h2" color={11} weight={450}>
          Sign In
        </Text.Text>
        <Flex.Box y full="x" empty>
          <Form.TextField path="username" inputProps={USERNAME_INPUT_PROPS} />
          <Form.TextField path="password" inputProps={PASSWORD_INPUT_PROPS} />
        </Flex.Box>
        <Flex.Box gap="small" align="center">
          <Flex.Box className={CSS.BE("login", "status")}>
            {stat.message !== "" && (
              <Status.Summary variant={stat.variant} message={stat.message} />
            )}
          </Flex.Box>
          <Button.Button
            onClick={handleSubmit}
            status={stat.variant === "loading" ? "loading" : undefined}
            trigger={SIGN_IN_TRIGGER}
            variant="filled"
            size="large"
          >
            Sign In
          </Button.Button>
        </Flex.Box>
      </Flex.Box>
    </Form.Form>
  );
};
