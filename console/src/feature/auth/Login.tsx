// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/auth/Login.css";

import { Logo } from "@synnaxlabs/media";
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
import { uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { LoginNav } from "@/feature/auth/LoginNav";
import { Cluster } from "@/platform/cluster";
import { CSS } from "@/platform/css";
import { Session } from "@/session";

const LOG_IN_TRIGGER: Triggers.Trigger = ["Enter"];

const credentialsZ = z.object({
  username: z.string().min(1, "Username is required"),
  password: z.string().min(1, "Password is required"),
});

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

export interface LoginProps {
  /** Renders the window nav chrome above the surface. Off in child windows. */
  nav?: boolean;
}

/**
 * Full-screen login surface: cluster selection plus credential entry. Serves
 * both initial login (no session selected) and credential re-entry when the
 * active cluster rejects auth, in which case the live connection status is
 * shown and submitting for the active cluster resumes its connection.
 */
export const Login = ({ nav = true }: LoginProps): ReactElement => {
  const client = Synnax.use();
  const connStatus = Synnax.useConnectionStatus();
  const servingCluster = Cluster.detectConnection();
  const clusters = Session.Cluster.useSelectMany();
  const activeKey = Session.Cluster.useSelectSelectedKey();
  const [selectedKey, setSelectedKey] = useState<string | undefined>(
    activeKey ?? clusters[0]?.key,
  );
  const selectedCluster = Session.Cluster.useSelectState(selectedKey);
  const dispatch = Session.useDispatch();

  const methods = Form.use<typeof credentialsZ>({
    schema: credentialsZ,
    values: { username: selectedCluster?.username ?? "", password: "" },
  });

  const handleSubmit = (): void => {
    const clusterToConnect = servingCluster ?? selectedCluster;
    if (!methods.validate() || clusterToConnect == null) return;
    const credentials = methods.value();
    const key =
      servingCluster == null && selectedCluster != null
        ? selectedCluster.key
        : (activeKey ?? uuid.create());
    // A same-credentials resubmit leaves the connection params untouched, so the
    // provider never swaps the client; nudge the existing one to reconnect.
    if (client != null && key === activeKey) client.reauthenticate(credentials);
    dispatch(Session.Cluster.set({ ...clusterToConnect, key, ...credentials }));
    dispatch(Session.Cluster.select(key));
  };

  const handleSelectedClusterChange = useCallback(
    (key?: string) => {
      if (key == null) return;
      const next = clusters.find((c) => c.key === key);
      methods.reset({ username: next?.username ?? "", password: "" });
      setSelectedKey(key);
    },
    [methods, clusters],
  );

  return (
    <Flex.Box y empty className={CSS.B("login")}>
      {nav && <LoginNav />}
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
                  {client != null && (
                    <Flex.Box className={CSS.BE("login", "status")}>
                      {connStatus.message !== "" && (
                        <Status.Summary
                          variant={connStatus.variant}
                          message={connStatus.message}
                        />
                      )}
                    </Flex.Box>
                  )}
                  <Button.Button
                    onClick={handleSubmit}
                    trigger={LOG_IN_TRIGGER}
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
