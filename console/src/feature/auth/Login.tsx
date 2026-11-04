// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/auth/Login.css";

import { type auth } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { type Triggers } from "@synnaxlabs/lyra/triggers";
import { Credential, Synnax } from "@synnaxlabs/pluto";
import { deep, errors } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef, useState } from "react";
import { z } from "zod";

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

const FAILURE_MESSAGES: Record<Core.Purpose, string> = {
  login: "Failed to log in",
  link: "Failed to link provider account",
  invite: "Failed to redeem invite code",
};

type ProviderCore = Parameters<typeof Core.loginDesktop>[0];

const assertAddressKept = (core: ProviderCore, current?: ProviderCore): void => {
  if (
    current?.host !== core.host ||
    current.port !== core.port ||
    current.secure !== core.secure
  )
    throw new Error(
      "The Core was removed or its address changed during the login. Log in again.",
    );
};

/**
 * Finishes a provider redirect, before any guard chooses between the login surface and
 * a session the browser already holds. A login replaces the session. A link keeps it,
 * and waits for the client to link the account to the logged-in user. An invite
 * replaces the session once the Core accepts the code.
 */
export const useCompleteBrowserLogin = (): void => {
  const client = Synnax.use();
  const cores = Session.Core.useSelectMany();
  const selected = Session.Core.useSelectSelectedKey();
  const servingCore = Core.detectConnection();
  const dispatch = Session.useDispatch();
  const store = Session.useStore();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const [pendingLink, setPendingLink] = useState<Core.CompletedLogin>();
  const link = Credential.OIDC.useLink({
    afterSuccess: ({ data }) =>
      addStatus({
        variant: "success",
        name: data.provider,
        message: "Linked provider account",
      }),
  });
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
        const saved = cores.find((c) => c.key === completed.core);
        const core = saved ?? servingCore;
        if (core == null) return;
        const { invite, credentials } = completed;
        const cleared = {
          username: "",
          password: "",
          oidc: undefined,
          token: undefined,
        };
        if (invite == null) {
          dispatch(Session.Core.set({ ...core, ...cleared, oidc: credentials }));
          dispatch(Session.Core.select(core.key));
          return;
        }
        handleError(async () => {
          const cookie = Core.servesPage(core);
          const token = await Core.redeemInvite(core, invite, credentials, cookie);
          const current = Session.Core.selectState(store.getState(), core.key);
          // a serving Core has no record until this login makes one
          if (saved != null) assertAddressKept(saved, current);
          dispatch(
            Session.Core.set({
              ...(current ?? core),
              ...cleared,
              token: cookie ? undefined : token,
              lastLogin: Date.now(),
            }),
          );
          dispatch(Session.Core.select(core.key));
        }, FAILURE_MESSAGES.invite);
      },
      FAILURE_MESSAGES[purpose ?? "login"],
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
      await link.updateAsync(pendingLink.credentials);
    }, "Failed to link provider account");
  }, [client, pendingLink]);
};

const startProviderLogin = async (
  core: ProviderCore,
  login: auth.OIDCLogin,
  invite?: string,
): Promise<void> => {
  if (Session.Runtime.ENGINE === "tauri") await Core.loginDesktop(core, login, invite);
  else if (invite == null) await Core.beginBrowserLogin(core.key, login);
  else await Core.beginBrowserInvite(core.key, login, invite);
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
  const store = Session.useStore();
  const [step, setStep] = useState<Step>(() =>
    servingCore != null || activeKey != null ? "login" : "cores",
  );
  const target = servingCore ?? selectedCore;
  const { password: passwordAccepted, oidc: oidcAccepted } =
    Core.useAcceptedMethods(target);
  const providers = Core.useProviders(target, oidcAccepted);
  const handleError = Status.useErrorHandler();
  // a second login would replace the first one's loopback listener
  const [pendingProvider, setPendingProvider] = useState<string>();
  const [inviteRevealed, setInviteRevealed] = useState(false);
  const [inviteCode, setInviteCode] = useState("");
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
    setPendingProvider(login.provider);
    const trimmed = inviteCode.trim();
    const invite = trimmed === "" ? undefined : trimmed;
    handleError(
      async () => {
        let failure: Error | undefined;
        try {
          await startProviderLogin(core, login, invite);
        } catch (e) {
          failure = errors.fromUnknown(e);
        }
        setPendingProvider(undefined);
        if (failure != null) throw failure;
        if (Session.Runtime.ENGINE !== "tauri") return;
        const current = Session.Core.selectState(store.getState(), core.key);
        // the token is saved under the address the login started with
        assertAddressKept(core, current);
        // a user who left the login surface has moved on
        if (mounted.current)
          dispatch(Session.Core.recordLogin({ key: core.key, at: Date.now() }));
      },
      invite == null
        ? `Failed to log in with ${login.provider}`
        : FAILURE_MESSAGES.invite,
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
    // A same-credentials resubmit leaves the connection params untouched, so the
    // provider never swaps the client; nudge the existing one to reconnect.
    if (client != null && key === activeKey) client.reauthenticate(credentials);
    dispatch(
      Session.Core.set({
        ...coreToConnect,
        ...credentials,
        oidc: undefined,
        token: undefined,
      }),
    );
    dispatch(Session.Core.select(key));
  };

  const handleSelectedCoreChange = (key?: string) => {
    if (key == null) return;
    const next = cores.find((c) => c.key === key);
    if (next == null) return;
    methods.reset({ username: next.username ?? "", password: "" });
    setInviteRevealed(false);
    setInviteCode("");
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
                  {inviteRevealed ? (
                    <Input.Text
                      value={inviteCode}
                      onChange={setInviteCode}
                      placeholder="Invite code"
                      size="large"
                      full="x"
                      autoFocus
                    />
                  ) : (
                    <Button.Button
                      variant="text"
                      full="x"
                      justify="center"
                      onClick={() => setInviteRevealed(true)}
                    >
                      I have an invite code
                    </Button.Button>
                  )}
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
