// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/cluster/LoginScreen.css";

import { z } from "zod";
import { id, status } from "@synnaxlabs/x";
import { Logo } from "@synnaxlabs/media";
import { Synnax as Client } from "@synnaxlabs/client";
import { Button, Flex, Form, Status } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";
import { set as setVersion } from "@/version/slice";
import { type ConnectionParams } from "@/cluster/autoConnect";
import { Cluster } from "@/cluster/types";

const loginSchema = z.object({
  username: z.string().min(1, "Username is required"),
  password: z.string().min(1, "Password is required"),
});

export interface Credentials {
  username: string;
  password: string;
}

export interface LoginScreenProps {
  connection: ConnectionParams;
  onSuccess: (cluster: Cluster) => Promise<void>;
}

export const LoginScreen = ({
  connection,
  onSuccess,
}: LoginScreenProps): ReactElement => {
  const [stat, setStatus] = useState<status.Status>(() =>
    status.create({
      variant: "disabled",
      message: "",
    }),
  );
  const dispatch = useDispatch();

  const methods = Form.use<typeof loginSchema>({
    schema: loginSchema,
    values: { username: "", password: "" },
    onChange: () => {
      const usernameField = methods.get("username");
      const passwordField = methods.get("password");
      setStatus(
        status.create({
          variant:
            usernameField.touched && passwordField.touched ? "success" : "disabled",
          message: "",
        }),
      );
    },
  });

  const handleSubmit = async (): Promise<void> => {
    if (!methods.validate()) return;
    const credentials = methods.value();
    setStatus(status.create({ variant: "loading", message: "Connecting..." }));
    const client = new Client({ ...connection, ...credentials });
    const state = await client.connectivity.check();
    if (state.status !== "connected") {
      setStatus(
        status.create({
          variant: "error",
          message: state.message ?? "Unknown error",
        }),
      );
      return;
    }
    if (state.nodeVersion != null) dispatch(setVersion(state.nodeVersion));
    await onSuccess({
      key: state.clusterKey,
      name: "Cluster",
      ...connection,
      ...credentials,
    });
  };

  return (
    <Flex.Box className="pluto-login-screen" center y>
      <Flex.Box className="pluto-login-container" y gap="huge">
        <Flex.Box y gap="small" align="center">
          <Logo variant="title" />
        </Flex.Box>
        <Form.Form<typeof loginSchema> {...methods}>
          <Flex.Box y empty align="center" grow>
            <Flex.Box y grow full="x" empty>
              <Form.TextField
                path="username"
                inputProps={{ placeholder: "synnax", autoFocus: true, size: "large" }}
              />
              <Form.TextField
                path="password"
                inputProps={{ placeholder: "seldon", type: "password", size: "large" }}
              />
            </Flex.Box>
            <Flex.Box style={{ height: "5rem" }}>
              {stat.message !== "" && (
                <Status.Summary variant={stat.variant} message={stat.message} />
              )}
            </Flex.Box>
            <Button.Button
              onClick={() => void handleSubmit()}
              status={stat.variant}
              trigger={["Enter"]}
              variant="filled"
              size="large"
            >
              Sign In
            </Button.Button>
          </Flex.Box>
        </Form.Form>
      </Flex.Box>
    </Flex.Box>
  );
};
