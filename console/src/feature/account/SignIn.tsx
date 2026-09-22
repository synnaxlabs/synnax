// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/account/SignIn.css";

import { Button, Flex, Icon, Status, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useState } from "react";

import { readName } from "@/feature/account/machine";
import { mintState, signInURL } from "@/feature/account/portal";
import { License } from "@/feature/license";
import { Shell } from "@/feature/shell";
import { CSS } from "@/platform/css";
import { Runtime } from "@/platform/runtime";
import { Shell as PlatformShell } from "@/platform/shell";
import { Session } from "@/session";

/** Follows whether the machine has a network to reach the portal over. */
const useOnline = (): boolean => {
  const [online, setOnline] = useState(navigator.onLine);
  useEffect(() => {
    const update = (): void => setOnline(navigator.onLine);
    window.addEventListener("online", update);
    window.addEventListener("offline", update);
    return () => {
      window.removeEventListener("online", update);
      window.removeEventListener("offline", update);
    };
  }, []);
  return online;
};

type Stage = "idle" | "waiting" | "file";

/**
 * Full-screen sign-in surface for Synnax Desktop. Sends the person to the portal in
 * their browser and waits for the link the portal opens the app with; a license file
 * is the fallback.
 */
export const SignIn = (): ReactElement => {
  const [stage, setStage] = useState<Stage>("idle");
  if (stage === "file")
    return <License.Activate standalone onBack={() => setStage("idle")} />;
  return <Browser stage={stage} onStage={setStage} />;
};

interface BrowserProps {
  stage: Exclude<Stage, "file">;
  onStage: (stage: Stage) => void;
}

const Browser = ({ stage, onStage }: BrowserProps): ReactElement => {
  const { email } = Session.Account.useSelect();
  const dispatch = Session.useDispatch();
  const handleError = Status.useErrorHandler();
  const { info } = License.useInfo();
  const version = Session.Version.use();
  const online = useOnline();

  const start = (): void =>
    handleError(async () => {
      if (info == null) return;
      const state = mintState();
      dispatch(Session.Account.beginSignIn(state));
      const url = signInURL({
        state,
        fingerprint: info.fingerprint,
        name: await readName(),
        version,
      });
      await Runtime.openExternal(url);
      onStage("waiting");
    }, "Failed to open the sign-in page");

  const lapsed = email != null;
  let content: ReactElement;
  if (!online)
    content = (
      <>
        <Status.Summary
          variant="warning"
          message="You are offline"
          description="Connect to the internet to sign in."
        />
        <Button.Button variant="filled" onClick={start} disabled={info == null}>
          <Icon.Refresh />
          Try again
        </Button.Button>
      </>
    );
  else if (stage === "waiting")
    content = (
      <>
        <Status.Summary
          variant="loading"
          message="Waiting for your browser..."
          description="Finish signing in there. This window updates on its own."
        />
        <Button.Button variant="outlined" onClick={start}>
          <Icon.Refresh />
          Try again
        </Button.Button>
      </>
    );
  else
    content = (
      <Button.Button variant="filled" onClick={start} disabled={info == null}>
        <Icon.OpenExternal />
        Sign in
      </Button.Button>
    );

  return (
    <Shell.Frame className={CSS.B("account-sign-in")}>
      <Flex.Box
        y
        align="center"
        gap="large"
        className={CSS.BE("account-sign-in", "body")}
      >
        <Status.Orbital core={<PlatformShell.Mark />} />
        <Flex.Box y align="center" gap="small">
          <Text.Text level="h4" weight={500}>
            {lapsed ? "Your sign-in has lapsed" : "Sign in to continue"}
          </Text.Text>
          <Text.Text level="p" color={10} align="center">
            {lapsed
              ? `Sign in again as ${email} to keep using Synnax Desktop.`
              : "Synnax Desktop links this computer to your Synnax account."}
          </Text.Text>
        </Flex.Box>
        {content}
        <Button.Button
          variant="text"
          size="small"
          onClick={() => onStage("file")}
          className={CSS.BE("account-sign-in", "file")}
        >
          Use a license file
        </Button.Button>
      </Flex.Box>
    </Shell.Frame>
  );
};
