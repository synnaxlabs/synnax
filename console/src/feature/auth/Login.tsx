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
  Icon,
  type Input,
  Status,
  Synnax,
  Text,
  type Triggers,
} from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { Cluster } from "@/platform/cluster";
import { CSS } from "@/platform/css";
import { Shell } from "@/platform/shell";
import { Session } from "@/session";

const LOG_IN_TRIGGER: Triggers.Trigger = ["Enter"];

// The surface is a stepped wizard: pick a cluster, then authenticate against
// it. Serving-cluster (browser) mode has no cluster step.
type Step = "clusters" | "login";

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

/**
 * Full-screen login surface: cluster selection plus credential entry. Serves
 * both initial login (no session selected) and credential re-entry when the
 * active cluster rejects auth, in which case the live connection status is
 * shown and submitting for the active cluster resumes its connection.
 */
export const Login = (): ReactElement => {
  const client = Synnax.use();
  const connStatus = Synnax.useConnectionStatus();
  const servingCluster = Cluster.detectConnection();
  const clusters = Session.Cluster.useSelectMany();
  const activeKey = Session.Cluster.useSelectSelectedKey();
  const [selectedKey, setSelectedKey] = useState<string | undefined>(activeKey);
  const selectedCluster = Session.Cluster.useSelectState(selectedKey);
  const dispatch = Session.useDispatch();
  const [step, setStep] = useState<Step>(() =>
    servingCluster != null || activeKey != null ? "login" : "clusters",
  );
  const target = servingCluster ?? selectedCluster;

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
      if (next == null) return;
      methods.reset({ username: next.username ?? "", password: "" });
      setSelectedKey(key);
      setStep("login");
    },
    [methods, clusters],
  );

  return (
    <Shell.Frame
      className={CSS(CSS.B("login"), CSS.M(`step-${step}`))}
      connection={step === "login" ? target : undefined}
    >
      {step === "clusters" ? (
        <Cluster.List
          className={CSS.BE("shell", "list")}
          value={undefined}
          onChange={handleSelectedClusterChange}
        />
      ) : (
        <Flex.Box y gap="huge" className={CSS.BE("login", "form")} grow>
          {servingCluster == null && (
            <Button.Button
              variant="text"
              className={CSS.BE("login", "back")}
              onClick={() => setStep("clusters")}
            >
              <Icon.Arrow.Left />
            </Button.Button>
          )}
          <Form.Form<typeof credentialsZ> {...methods}>
            <Flex.Box y align="center" justify="center" grow gap="huge" shrink={false}>
              <Flex.Box center grow={false} className={CSS.BE("shell", "mark-ring")}>
                <Logo variant="icon" className={CSS.BE("shell", "mark")} />
              </Flex.Box>
              {target != null && (
                <Flex.Box
                  y
                  align="center"
                  gap="tiny"
                  grow={false}
                  className={CSS.BE("login", "target")}
                >
                  <Text.Text color={10} weight={500}>
                    {target.name}
                  </Text.Text>
                  <Text.Text level="small" color={9}>
                    {target.host}:{target.port}
                  </Text.Text>
                </Flex.Box>
              )}
              <Flex.Box y full="x" empty>
                <Form.TextField
                  path="username"
                  required={false}
                  inputProps={USERNAME_INPUT_PROPS}
                />
                <Form.TextField
                  path="password"
                  required={false}
                  inputProps={PASSWORD_INPUT_PROPS}
                />
              </Flex.Box>
              <Flex.Box gap="small" align="center" full="x">
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
                  full="x"
                  justify="center"
                >
                  Log In
                </Button.Button>
              </Flex.Box>
            </Flex.Box>
          </Form.Form>
        </Flex.Box>
      )}
    </Shell.Frame>
  );
};
