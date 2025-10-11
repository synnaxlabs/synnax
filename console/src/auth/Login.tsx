// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/cluster/LoginScreen.css";

import { Synnax as Client } from "@synnaxlabs/client";
import { Logo } from "@synnaxlabs/media";
import { Button, Flex, Form, Status, type Triggers } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { type ConnectionParams } from "@/cluster/detectConnection";
import { type Cluster } from "@/cluster/types";
import { Layouts } from "@/layouts";
import { set as setVersion } from "@/version/slice";

const SIGN_IN_TRIGGER: Triggers.Trigger = ["Enter"];

const credentialsZ = z.object({
  username: z.string().min(1, "Username is required"),
  password: z.string().min(1, "Password is required"),
});

export interface Credentials extends z.infer<typeof credentialsZ> {}

export interface LoginProps {
  connection: ConnectionParams;
  onSuccess: (cluster: Cluster) => void;
}

export const Login = ({ connection, onSuccess }: LoginProps): ReactElement => {
  const [stat, setStatus] = useState<status.Status>(() =>
    status.create({ variant: "disabled", message: "" }),
  );
  const dispatch = useDispatch();
  const handleError = Status.useErrorHandler();

  const methods = Form.use<typeof credentialsZ>({
    schema: credentialsZ,
    values: { username: "", password: "" },
    onChange: () => {
      const usernameTouched = methods.get("username").touched;
      const passwordTouched = methods.get("password").touched;
      setStatus(
        status.create({
          variant: usernameTouched && passwordTouched ? "success" : "disabled",
          message: "",
        }),
      );
    },
  });

  const handleSubmit = (): void =>
    handleError(async () => {
      if (!methods.validate()) return;
      const credentials = methods.value();
      setStatus(status.create({ variant: "loading", message: "Connecting..." }));
      const client = new Client({ ...connection, ...credentials });
      const state = await client.connectivity.check();
      if (state.status !== "connected") {
        const message = state.message ?? "Unknown error";
        return setStatus(status.create({ variant: "error", message }));
      }
      // Use the cluster's version as the console version on web.
      if (state.nodeVersion != null) dispatch(setVersion(state.nodeVersion));
      onSuccess({
        key: state.clusterKey,
        name: "Core",
        ...connection,
        ...credentials,
      });
    }, "Failed to sign in");

  return (
    <>
      <Layouts.Notifications />
      <Flex.Box className="pluto-login-screen" center y>
        <Flex.Box className="pluto-login-container" y gap="huge">
          <Flex.Box y gap="small" align="center">
            <Logo variant="title" />
          </Flex.Box>
          <Form.Form<typeof credentialsZ> {...methods}>
            <Flex.Box y empty align="center" grow>
              <Flex.Box y grow full="x" empty>
                <Form.TextField
                  path="username"
                  inputProps={{ placeholder: "synnax", autoFocus: true, size: "large" }}
                />
                <Form.TextField
                  path="password"
                  inputProps={{
                    placeholder: "seldon",
                    type: "password",
                    size: "large",
                  }}
                />
              </Flex.Box>
              <Flex.Box style={{ height: "5rem" }}>
                {stat.message !== "" && (
                  <Status.Summary variant={stat.variant} message={stat.message} />
                )}
              </Flex.Box>
              <Button.Button
                onClick={handleSubmit}
                status={stat.variant}
                trigger={SIGN_IN_TRIGGER}
                variant="filled"
                size="large"
              >
                Sign In
              </Button.Button>
            </Flex.Box>
          </Form.Form>
        </Flex.Box>
      </Flex.Box>
    </>
  );
};
