// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/account/Login.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useEffect, useState } from "react";

import { loginURL, mintState } from "@/feature/account/handoff";
import { readMachineName } from "@/feature/account/machine";
import { License } from "@/feature/license";
import { Shell } from "@/feature/shell";
import { CSS } from "@/platform/css";
import { Runtime } from "@/platform/runtime";
import { Shell as PlatformShell } from "@/platform/shell";
import { Session } from "@/session";

/** Follows whether the machine has a network to reach the hub over. */
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
 * Full-screen login surface for Synnax Desktop. Sends the person to the hub in
 * their browser and waits for the link the hub opens the app with; a license file
 * is the fallback.
 */
export const Login = (): ReactElement => {
  const [stage, setStage] = useState<Stage>("idle");
  if (stage === "file") return <License.Activate onBack={() => setStage("idle")} />;
  return <Handoff stage={stage} onStage={setStage} />;
};

interface HandoffProps {
  stage: Exclude<Stage, "file">;
  onStage: (stage: Stage) => void;
}

const Handoff = ({ stage, onStage }: HandoffProps): ReactElement => {
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
      dispatch(Session.Account.beginLogin(state));
      const url = loginURL({
        state,
        fingerprint: info.fingerprint,
        name: await readMachineName(),
        version,
      });
      await Runtime.openExternal(url);
      onStage("waiting");
    }, "Failed to open the login page");

  const lapsed = email != null;
  let title: string;
  let description: string;
  let action: ReactElement;
  if (!online) {
    title = "You are offline";
    description = "Connect to the internet to log in";
    action = (
      <Button.Button
        size="large"
        full="x"
        justify="center"
        variant="filled"
        onClick={start}
        disabled={info == null}
      >
        <Icon.Refresh />
        Try again
      </Button.Button>
    );
  } else if (stage === "waiting") {
    title = "Finish in your browser";
    description = "This window updates on its own once you log in";
    action = (
      <Button.Button
        size="large"
        full="x"
        justify="center"
        variant="outlined"
        onClick={start}
      >
        Open the page again
        <Icon.OpenExternal />
      </Button.Button>
    );
  } else {
    title = lapsed ? "Your login has lapsed" : "Log in to continue";
    description = lapsed
      ? `Log in again as ${email} to keep using Synnax Desktop`
      : "Link this computer to your Synnax account";
    action = (
      <Button.Button
        size="large"
        full="x"
        justify="center"
        variant="filled"
        onClick={start}
        disabled={info == null}
      >
        Log in
        <Icon.OpenExternal />
      </Button.Button>
    );
  }

  return (
    <Shell.Frame className={CSS.B("account-login")}>
      <Flex.Box y align="center" className={CSS.BE("account-login", "body")}>
        <Flex.Box y align="center" justify="center" gap="huge" grow full="x">
          <Flex.Box
            y
            align="center"
            justify="center"
            className={CSS.BE("account-login", "hero")}
          >
            {stage === "waiting" && online ? (
              <Status.Orbital core={<PlatformShell.Mark />} />
            ) : (
              <PlatformShell.Mark />
            )}
          </Flex.Box>
          <Flex.Box y align="center" gap="small" full="x">
            <Text.Text level="h4" weight={500} color={11}>
              {title}
            </Text.Text>
            <Text.Text
              level="p"
              color={9}
              className={CSS.BE("account-login", "description")}
            >
              {description}
            </Text.Text>
          </Flex.Box>
          {action}
        </Flex.Box>
        <Button.Button
          variant="text"
          size="small"
          textColor={9}
          onClick={() => onStage("file")}
        >
          Use a license file
        </Button.Button>
      </Flex.Box>
    </Shell.Frame>
  );
};
