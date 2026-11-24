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
  type auth,
  type connection,
  ExpiredTokenError,
  InvalidCredentialsError,
  InvalidTokenError,
  MissingCredentialsError,
  SessionEndedError,
} from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import type { Input } from "@synnaxlabs/lyra/input";
import { Note } from "@synnaxlabs/lyra/note";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import type { Triggers } from "@synnaxlabs/lyra/triggers";
import { Credential, Synnax } from "@synnaxlabs/pluto";
import { deep, errors } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef, useState } from "react";
import { z } from "zod";

import { display } from "@/feature/auth/brand";
import { Shell } from "@/feature/shell";
import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Shell as PlatformShell } from "@/platform/shell";
import { Session } from "@/session";

const path = deep.path<z.infer<typeof credentialsZ>>();

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

/** Says why the Core refused a login, or undefined when no login failed. */
const errorMessage = (status: connection.Status): string | undefined => {
  if (status.variant !== "error" || status.details.reason !== "auth") return undefined;
  const { error } = status.details;
  if (error == null || MissingCredentialsError.matches(error)) return undefined;
  if (InvalidCredentialsError.matches(error)) return "Incorrect username or password.";
  if (
    InvalidTokenError.matches(error) ||
    ExpiredTokenError.matches(error) ||
    SessionEndedError.matches(error)
  )
    return "Your session ended. Log in again.";
  return error.message;
};

/**
 * Finishes a provider redirect, before any guard chooses between the login surface and
 * a session the browser already holds. A login replaces the session. A link keeps it,
 * and waits for the client to link the account to the logged-in user.
 */
export const useCompleteBrowserLogin = (): void => {
  const client = Synnax.use();
  const cores = Session.Core.useSelectMany();
  const selected = Session.Core.useSelectSelectedKey();
  const servingCore = Core.detectConnection();
  const dispatch = Session.useDispatch();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const [pendingLink, setPendingLink] = useState<Core.CompletedLogin>();
  const link = Credential.OIDC.useLink();
  useEffect(() => {
    const purpose = Core.pendingPurpose();
    handleError(
      () => {
        const completed = Core.completeBrowserLogin();
        if (completed == null) return;
        if (completed.purpose === "link") {
          if (completed.core !== selected)
            throw new Error(
              "The Console left the Core that started the link. Log in and try again.",
            );
          setPendingLink(completed);
          return;
        }
        const core = cores.find((c) => c.key === completed.core) ?? servingCore;
        if (core == null) return;
        const oidc = completed.credentials;
        dispatch(Session.Core.set({ ...core, username: "", password: "", oidc }));
        dispatch(Session.Core.select(core.key));
      },
      purpose === "link" ? "Failed to link provider account" : "Failed to log in",
    );
  }, []);
  useEffect(() => {
    if (pendingLink == null || client == null) return;
    setPendingLink(undefined);
    handleError(async () => {
      // the session cookie is shared, so another tab can change the user mid-redirect
      await client.auth.sessions.retrieve();
      if (client.auth.subject?.key !== pendingLink.subject)
        throw new Error("Another user logged in during the link. Please try again.");
      const { provider } = pendingLink.credentials;
      if (await link.updateAsync(pendingLink.credentials))
        addStatus({
          variant: "success",
          name: provider,
          message: "Linked provider account",
        });
    }, "Failed to link provider account");
  }, [client, pendingLink]);
};

/**
 * Full-screen login surface. Serves both initial login and credential re-entry after
 * the active Core rejects auth. Each submit is a new login, with a new client.
 */
export const Login = (): ReactElement => {
  const message = errorMessage(Synnax.useConnectionStatus());
  const servingCore = Core.detectConnection();
  const cores = Session.Core.useSelectMany();
  const activeKey = Session.Core.useSelectSelectedKey();
  const [selectedKey, setSelectedKey] = useState<string | undefined>(activeKey);
  const selectedCore = Session.Core.useSelectState(selectedKey);
  const dispatch = Session.useDispatch();
  const store = Session.useStore();
  const [step, setStep] = useState<Step>(() =>
    servingCore != null || activeKey != null ? "login" : "cores",
  );
  const target = servingCore ?? selectedCore;
  const { password: passwordAccepted, oidc: oidcAccepted } =
    Core.useAcceptedMethods(target);
  const loadedProviders = Core.useProviders(target, oidcAccepted);
  const providers = loadedProviders ?? [];
  // a provider list that is still loading, or failed to load, says nothing yet
  const noLogin = !passwordAccepted && (!oidcAccepted || loadedProviders?.length === 0);
  const handleError = Status.useErrorHandler();
  const reportUnfinished = Core.useReportUnfinished();
  // a newer provider login replaces an older one, which then ends unreported
  const attempts = useRef(0);
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const handleProvider = (login: auth.OIDCLogin): void => {
    const core = servingCore ?? selectedCore;
    if (core == null) return;
    const attempt = ++attempts.current;
    handleError(
      async () => {
        let failure: Error | undefined;
        try {
          if (Session.Runtime.ENGINE !== "tauri")
            await Core.beginBrowserLogin(core.key, login);
          else await Core.loginDesktop(core, login);
        } catch (e) {
          failure = errors.fromUnknown(e);
        }
        if (attempt !== attempts.current) return;
        if (failure != null) {
          reportUnfinished(failure, "Login not finished");
          return;
        }
        if (Session.Runtime.ENGINE !== "tauri") return;
        const current = Session.Core.selectState(store.getState(), core.key);
        // the token is saved under the address the login started with
        if (
          current?.host !== core.host ||
          current.port !== core.port ||
          current.secure !== core.secure
        )
          throw new Error("The Core's address changed during the login. Log in again.");
        // a user who left the login surface has moved on
        if (mounted.current)
          dispatch(Session.Core.recordLogin({ key: core.key, at: Date.now() }));
      },
      `Failed to log in with ${display(login).label}`,
    );
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
    dispatch(
      Session.Core.set({
        ...coreToConnect,
        ...credentials,
        oidc: undefined,
        lastLogin: Date.now(),
      }),
    );
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

  const errorNote = (
    <Note.Note
      variant="error"
      full="x"
      className={CSS.cls(CSS.BE("login", "error"), message == null && CSS.M("ghost"))}
    >
      <Text.Text level="small">{message ?? "\u00a0"}</Text.Text>
    </Note.Note>
  );
  return (
    <Shell.Frame
      className={CSS.cls(
        CSS.B("login"),
        CSS.M(`step-${step}`),
        step === "login" &&
          passwordAccepted &&
          providers.length > 0 &&
          window.isSecureContext &&
          CSS.M("providers"),
      )}
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
            <Flex.Box
              y
              align="center"
              justify="center"
              grow
              gap="huge"
              className={CSS.BE("login", "content")}
            >
              <PlatformShell.Mark />
              <Flex.Box y align="center" gap="medium" full="x">
                {passwordAccepted ? (
                  <>
                    <Flex.Box y full="x" empty>
                      <Form.TextField
                        path={path.username}
                        required={false}
                        inputProps={USERNAME_INPUT_PROPS}
                      />
                      <Form.TextField
                        path={path.password}
                        required={false}
                        inputProps={PASSWORD_INPUT_PROPS}
                      />
                    </Flex.Box>
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
                  </>
                ) : (
                  noLogin && (
                    <Flex.Box y align="center" gap="small">
                      <Text.Text level="h4" status="warning">
                        <Status.Indicator variant="warning" />
                        No way to log in
                      </Text.Text>
                      <Text.Text level="p" color={9} style={{ textAlign: "center" }}>
                        This Core accepts neither a password nor a provider login. Ask
                        an administrator to turn on password login or to add a provider.
                      </Text.Text>
                    </Flex.Box>
                  )
                )}
                {passwordAccepted && providers.length > 0 && (
                  <Flex.Box x align="center" gap="small" full="x">
                    <Divider.Divider x grow />
                    <Text.Text level="small" color={8}>
                      or
                    </Text.Text>
                    <Divider.Divider x grow />
                  </Flex.Box>
                )}
                {providers.length > 0 && !window.isSecureContext && (
                  <Status.Summary
                    variant="warning"
                    message="Provider login needs HTTPS"
                    description="The browser can only log in through a provider on a page served over HTTPS. Open the Console over HTTPS, or use the desktop Console."
                  />
                )}
                {providers.length > 0 && window.isSecureContext && (
                  <Flex.Box
                    y
                    gap="small"
                    full="x"
                    className={CSS.BE("login", "providers")}
                  >
                    <Icon.Colored>
                      {providers.map((login) => {
                        const { label, icon } = display(login);
                        return (
                          <Button.Button
                            key={login.provider}
                            onClick={() => handleProvider(login)}
                            variant="outlined"
                            size="large"
                            full="x"
                            justify="center"
                          >
                            {icon}
                            Log in with {label}
                          </Button.Button>
                        );
                      })}
                    </Icon.Colored>
                  </Flex.Box>
                )}
                {errorNote}
              </Flex.Box>
            </Flex.Box>
          </Form.Form>
        </Flex.Box>
      )}
    </Shell.Frame>
  );
};
