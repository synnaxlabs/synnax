// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/auth/Login.css";

import { auth } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { type Triggers } from "@synnaxlabs/lyra/triggers";
import { Synnax } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef, useState } from "react";
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

interface Accepted {
  password: boolean;
  oidc: boolean;
  providers: auth.OIDCLogin[];
}

const DEFAULT_ACCEPTED: Accepted = { password: true, oidc: false, providers: [] };

// An unreachable Core, or one older than the methods endpoint, keeps the password
// form: the connection status reports the first, and the second accepts passwords.
const useAccepted = (core?: auth.MethodsParams): Accepted => {
  const [accepted, setAccepted] = useState<Accepted>(DEFAULT_ACCEPTED);
  const handleError = Status.useErrorHandler();
  useAsyncEffect(
    async (signal) => {
      setAccepted(DEFAULT_ACCEPTED);
      if (core == null) return;
      const methods = await auth.retrieveMethods(core).catch(() => null);
      if (methods == null || signal.aborted) return;
      const oidc = methods.includes("oidc");
      const password = methods.includes("password");
      setAccepted({ password, oidc, providers: [] });
      if (!oidc) return;
      handleError(async () => {
        const providers = await auth.retrieveProviders(core);
        if (!signal.aborted) setAccepted({ password, oidc, providers });
      }, "Failed to retrieve the login providers");
    },
    [core?.host, core?.port, core?.secure],
  );
  return accepted;
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
  const {
    password: passwordAccepted,
    oidc: oidcAccepted,
    providers,
  } = useAccepted(target);
  const handleError = Status.useErrorHandler();
  // a second login would replace the first one's loopback listener
  const [pendingProvider, setPendingProvider] = useState<string>();
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const connect = (
    core: Pick<Session.Core.Core, "host" | "key" | "name" | "port" | "secure">,
    oidc: auth.OIDCCredentials,
  ): void => {
    dispatch(Session.Core.set({ ...core, username: "", password: "", oidc }));
    dispatch(Session.Core.select(core.key));
  };

  // the browser Console comes back here from its provider with a code to log in with
  useEffect(() => {
    handleError(() => {
      const completed = Core.completeBrowserLogin();
      if (completed == null) return;
      const core = cores.find((c) => c.key === completed.core) ?? servingCore;
      if (core != null) connect(core, completed.credentials);
    }, "Failed to log in");
  }, []);

  const handleProvider = (login: auth.OIDCLogin): void => {
    const core = servingCore ?? selectedCore;
    if (core == null) return;
    setPendingProvider(login.provider);
    handleError(async () => {
      let failure: Error | undefined;
      try {
        if (Session.Runtime.ENGINE !== "tauri")
          await Core.beginBrowserLogin(core.key, login);
        // the desktop Console has no serving Core, so core is the selected record
        else if (selectedCore != null) {
          await Core.loginDesktop(selectedCore, login);
          // a user who left the login surface has moved on
          if (mounted.current)
            dispatch(Session.Core.recordLogin({ key: core.key, at: Date.now() }));
        }
      } catch (e) {
        failure = errors.fromUnknown(e);
      }
      setPendingProvider(undefined);
      if (failure != null) throw failure;
    }, `Failed to log in with ${login.provider}`);
  };

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
    dispatch(Session.Core.set({ ...coreToConnect, ...credentials, oidc: undefined }));
    dispatch(Session.Core.select(key));
  };

  const handleSelectedCoreChange = (key?: string) => {
    if (key == null) return;
    const next = cores.find((c) => c.key === key);
    if (next == null) return;
    methods.reset({ username: next.username ?? "", password: "" });
    setSelectedKey(key);
    setStep("login");
  };

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
              {passwordAccepted ? (
                <>
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
                            description={connStatus.description}
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
                </>
              ) : (
                !oidcAccepted && (
                  <Status.Summary
                    variant="warning"
                    message="Password login is off"
                    description="This Core does not accept passwords. Ask an administrator to add password to its accepted login methods."
                  />
                )
              )}
              {providers.length > 0 && !window.isSecureContext && (
                <Status.Summary
                  variant="warning"
                  message="Provider login needs HTTPS"
                  description="The browser can only log in through a provider on a page served over HTTPS. Open the Console over HTTPS, or use the desktop Console."
                />
              )}
              {providers.length > 0 && window.isSecureContext && (
                <Flex.Box y gap="small" full="x">
                  {providers.map((login) => (
                    <Button.Button
                      key={login.provider}
                      onClick={() => handleProvider(login)}
                      disabled={pendingProvider != null}
                      variant="outlined"
                      size="large"
                      full="x"
                      justify="center"
                    >
                      Log in with {login.provider}
                    </Button.Button>
                  ))}
                </Flex.Box>
              )}
            </Flex.Box>
          </Form.Form>
        </Flex.Box>
      )}
    </Shell.Frame>
  );
};
