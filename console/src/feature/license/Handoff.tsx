// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { TimeSpan, uuid } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useEffect, useState } from "react";

import { License } from "@/platform/license";

// The Console asks the portal for the license key often at first, while the user is
// likely on the portal, then slowly while it waits on someone the link was sent to.
const FAST_INTERVAL = TimeSpan.seconds(2);
const FAST_PERIOD = TimeSpan.minutes(10);
const SLOW_INTERVAL = TimeSpan.minutes(1);

type State =
  | { variant: "idle" }
  | { variant: "waiting" }
  | { variant: "activated" }
  | { variant: "failed"; status: status.Status };

export interface HandoffProps {
  fingerprint: string[];
  /** The label of the button that opens the portal. */
  children: ReactNode;
  variant?: Button.Variant;
}

/**
 * A button that opens the portal page that issues this Core a license key. Once the
 * page opens, or its link is copied, the Console waits for the key and activates it on
 * the Core when the portal hands it back.
 */
export const Handoff = ({
  fingerprint,
  children,
  variant = "filled",
}: HandoffProps): ReactElement => {
  const client = Synnax.use();
  const [code] = useState(uuid.create);
  const [state, setState] = useState<State>({ variant: "idle" });
  const waiting = state.variant === "waiting";
  const url = License.activateURL({ fingerprint, handoff: code });
  useEffect(() => {
    if (!waiting || client == null) return;
    const controller = new AbortController();
    const poll = async (): Promise<void> => {
      let key: string | null;
      try {
        key = await License.receive(code, controller.signal);
      } catch (e) {
        if (controller.signal.aborted) return;
        // An unreachable portal is normal on a machine with no internet access. The
        // key field stays open, and the next poll tries again.
        if (e instanceof TypeError) return;
        setState({
          variant: "failed",
          status: status.fromException(e, "Failed to get the license key"),
        });
        return;
      }
      if (key == null || controller.signal.aborted) return;
      controller.abort();
      try {
        await client.license.activate(key);
        setState({ variant: "activated" });
      } catch (e) {
        setState({
          variant: "failed",
          status: status.fromException(e, "Failed to activate license"),
        });
      }
    };
    let waited = TimeSpan.ZERO;
    let timer: ReturnType<typeof setTimeout>;
    const next = (): void => {
      const interval = waited.lessThan(FAST_PERIOD) ? FAST_INTERVAL : SLOW_INTERVAL;
      waited = waited.add(interval);
      timer = setTimeout(() => void tick(), interval.milliseconds);
    };
    const tick = async (): Promise<void> => {
      await poll();
      if (!controller.signal.aborted) next();
    };
    next();
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, [waiting, client, code]);
  const wait = (): void => setState({ variant: "waiting" });
  let line: ReactElement;
  switch (state.variant) {
    case "idle":
      line = (
        <Text.Text level="small" color={9}>
          Someone else manages your licenses?
        </Text.Text>
      );
      break;
    case "waiting":
      line = (
        <Status.Summary
          level="small"
          variant="loading"
          message="Waiting for the portal. This updates on its own."
        />
      );
      break;
    case "activated":
      line = (
        <Status.Summary
          level="small"
          variant="success"
          message="Activated the license"
        />
      );
      break;
    case "failed":
      line = <Status.Summary level="small" status={state.status} />;
  }
  return (
    <Flex.Box y gap="small" full="x">
      <Button.Button
        variant={variant}
        size="large"
        full="x"
        justify="center"
        href={url}
        target="_blank"
        onClick={wait}
      >
        {children}
        <Icon.OpenExternal />
      </Button.Button>
      <Flex.Box x align="center" justify="center" gap="small">
        {line}
        <Button.Copy
          variant="text"
          size="small"
          textColor={9}
          text={url}
          onCopy={wait}
          successMessage="Copied activation link"
        >
          Copy link
        </Button.Copy>
      </Flex.Box>
    </Flex.Box>
  );
};
