// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/auth/Login.css";

import { Synnax as Client } from "@synnaxlabs/client";
import { Logo } from "@synnaxlabs/media";
import {
  Button,
  Flex,
  Form,
  type Input,
  Status,
  Text,
  type Triggers,
} from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { LoginNav } from "@/auth/LoginNav";
import { Cluster } from "@/cluster";
import { CSS } from "@/css";
import { Version } from "@/version";

const SIGN_IN_TRIGGER: Triggers.Trigger = ["Enter"];

const credentialsZ = z.object({
  username: z.string().min(1, "Username is required"),
  password: z.string().min(1, "Password is required"),
});

export interface Credentials extends z.infer<typeof credentialsZ> {}

const USERNAME_INPUT_PROPS: Partial<Input.TextProps> = {
  placeholder: "synnax",
  autoFocus: true,
  size: "large",
};

const PASSWORD_INPUT_PROPS: Partial<Input.TextProps> = {
  placeholder: "seldon",
  type: "password",
  size: "large",
};

export const Login = (): ReactElement => {
  const servingCluster = Cluster.detectConnection();
  const [stat, setStatus] = useState<status.Status>(() =>
    status.create({ variant: "disabled", message: "" }),
  );
  const clusters = Cluster.useSelectMany();
  const [selectedKey, setSelectedKey] = useState<string | undefined>(clusters[0]?.key);
  const selectedCluster = Cluster.useSelect(selectedKey);
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
      const clusterToConnect = servingCluster ?? selectedCluster;
      if (!methods.validate() || clusterToConnect == null) return;
      const credentials = methods.value();
      setStatus(status.create({ variant: "loading", message: "Connecting..." }));
      const client = new Client({ ...clusterToConnect, ...credentials });
      const state = await client.connectivity.check();
      const key = state.clusterKey;
      if (state.status !== "connected") {
        const message = state.message ?? "Unknown error";
        return setStatus(status.create({ variant: "error", message }));
      }
      if (state.nodeVersion != null && servingCluster != null)
        dispatch(Version.set(state.nodeVersion));
      dispatch(Cluster.set({ ...clusterToConnect, key, ...credentials }));
      dispatch(Cluster.setActive(key));
    }, "Failed to log in");

  const handleSelectedClusterChange = useCallback(
    (key?: string) => {
      if (key == null) return;
      methods.reset();
      setSelectedKey(key);
    },
    [methods],
  );

  return (
    <Flex.Box y empty className={CSS.B("login")}>
      <LoginNav />
      <Flex.Box
        y
        align="center"
        justify="center"
        background={1}
        gap="huge"
        grow
        data-tauri-drag-region
        className={CSS.BE("login", "content")}
      >
        <Logo
          variant="title"
          className={CSS.BE("login", "logo")}
          data-tauri-drag-region
        />
        <Flex.Box
          pack
          x
          className={CSS(
            CSS.BE("login", "container"),
            servingCluster != null && CSS.M("narrow"),
          )}
          grow={false}
          rounded={1.5}
          background={0}
        >
          {servingCluster == null && (
            <Cluster.List
              className={CSS.BE("login", "list")}
              value={selectedKey}
              onChange={handleSelectedClusterChange}
            />
          )}
          <Flex.Box
            y
            gap="huge"
            className={CSS.BE("login", "form")}
            bordered
            grow
            shrink={false}
          >
            <Form.Form<typeof credentialsZ> {...methods}>
              <Flex.Box y align="center" grow gap="huge" shrink={false}>
                <Text.Text level="h2" color={11} weight={450}>
                  Log In
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
                    status={stat.variant}
                    trigger={SIGN_IN_TRIGGER}
                    variant="filled"
                    size="large"
                  >
                    Log In
                  </Button.Button>
                </Flex.Box>
              </Flex.Box>
            </Form.Form>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};
