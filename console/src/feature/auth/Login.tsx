// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/auth/Login.css";

import {
  Button,
  Flex,
  Form,
  Icon,
  type Input,
  Status,
  Synnax,
  type Triggers,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { Shell } from "@/feature/shell";
import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Shell as PlatformShell } from "@/platform/shell";
import { Session } from "@/session";

const LOG_IN_TRIGGER: Triggers.Trigger = ["Enter"];

type Step = "cores" | "login";

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
 * Full-screen login surface. Serves both initial login and credential re-entry after
 * the active Core rejects auth, where submitting resumes its connection.
 */
export const Login = (): ReactElement => {
  const client = Synnax.use();
  const connStatus = Synnax.useConnectionStatus();
  const servingCore = Core.detectConnection();
  const cores = Session.Core.useSelectMany();
  const activeKey = Session.Core.useSelectSelectedKey();
  const [selectedKey, setSelectedKey] = useState<string | undefined>(activeKey);
  const selectedCore = Session.Core.useSelectState(selectedKey);
  const dispatch = Session.useDispatch();
  const [step, setStep] = useState<Step>(() =>
    servingCore != null || activeKey != null ? "login" : "cores",
  );
  const target = servingCore ?? selectedCore;

  const methods = Form.use<typeof credentialsZ>({
    schema: credentialsZ,
    values: { username: selectedCore?.username ?? "", password: "" },
  });

  const handleSubmit = (): void => {
    const coreToConnect = servingCore ?? selectedCore;
    if (!methods.validate() || coreToConnect == null) return;
    const credentials = methods.value();
    const { key } = coreToConnect;
    // A same-credentials resubmit leaves the connection params untouched, so the
    // provider never swaps the client; nudge the existing one to reconnect.
    if (client != null && key === activeKey) client.reauthenticate(credentials);
    dispatch(Session.Core.set({ ...coreToConnect, ...credentials }));
    dispatch(Session.Core.select(key));
  };

  const handleSelectedCoreChange = useCallback(
    (key?: string) => {
      if (key == null) return;
      const next = cores.find((c) => c.key === key);
      if (next == null) return;
      methods.reset({ username: next.username ?? "", password: "" });
      setSelectedKey(key);
      setStep("login");
    },
    [methods, cores],
  );

  return (
    <Shell.Frame
      className={CSS.cls(CSS.B("login"), CSS.M(`step-${step}`))}
      connection={step === "login" ? target : undefined}
    >
      {step === "cores" ? (
        <Core.List
          className={CSS.BE("shell", "list")}
          value={undefined}
          onChange={handleSelectedCoreChange}
        />
      ) : (
        <Flex.Box y gap="huge" className={CSS.BE("login", "form")} grow>
          {servingCore == null && (
            <Button.Button
              variant="text"
              className={CSS.BE("login", "back")}
              onClick={() => setStep("cores")}
            >
              <Icon.Arrow.Left />
            </Button.Button>
          )}
          <Form.Form<typeof credentialsZ> {...methods}>
            <Flex.Box y align="center" justify="center" grow gap="huge" shrink={false}>
              <PlatformShell.Mark />
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
                  Log in
                  <Icon.Arrow.Right />
                </Button.Button>
              </Flex.Box>
            </Flex.Box>
          </Form.Form>
        </Flex.Box>
      )}
    </Shell.Frame>
  );
};
