// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/license/Activate.css";

import { ExpiredLicenseError } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { License as PLicense, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Handoff } from "@/feature/license/Handoff";
import { KeyInput } from "@/feature/license/KeyInput";
import { Shell } from "@/feature/shell";
import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Shell as PlatformShell } from "@/platform/shell";
import { Session } from "@/session";

export interface ActivateProps {
  /** Offers a way back to the screen that led here. */
  onBack?: () => void;
}

/**
 * Full-screen activation surface for a Core that refuses requests until a license
 * applies. The connection check keeps polling, so the screen leaves on its own once a
 * license applies from anywhere.
 */
export const Activate = ({ onBack }: ActivateProps): ReactElement => {
  const { details } = Synnax.useConnectionStatus();
  const target = Session.Core.useSelectSelected();
  const logout = Core.useLogout();
  const result = PLicense.useResult({});
  const fingerprint = result.data?.fingerprint ?? [];
  const expired = ExpiredLicenseError.matches(details.error);
  const noHardware = result.data != null && fingerprint.length === 0;

  let description =
    "Activate it on the Synnax portal. The license key comes back here on its own.";
  if (noHardware)
    description =
      "This Core reports no network hardware to bind a license to. Ask Synnax Labs " +
      "for a floating license, then paste it below.";
  else if (expired)
    description =
      "Get a new license key on the Synnax portal. It comes back here on its own.";

  return (
    <Shell.Frame
      className={CSS.B("license-activate")}
      connection={Session.Runtime.CORE_EMBEDDED ? null : target}
    >
      <Flex.Box y align="center" className={CSS.BE("license-activate", "body")}>
        <Flex.Box y align="center" justify="center" gap="huge" full="x" grow>
          <PlatformShell.Mark />
          <Flex.Box y align="center" gap="small" full="x">
            <Text.Text level="h4" weight={500} color={11}>
              {expired
                ? "This Core's license has expired"
                : "This Core needs a license"}
            </Text.Text>
            <Text.Text
              level="p"
              color={9}
              className={CSS.BE("license-activate", "description")}
            >
              {description}
            </Text.Text>
          </Flex.Box>
          <Flex.Box y gap="huge" full="x">
            {result.data != null && !noHardware && (
              <Handoff fingerprint={fingerprint}>
                {expired ? "Get a new key on the portal" : "Activate on the portal"}
              </Handoff>
            )}
            <Flex.Box y gap="small" full="x">
              <Text.Text level="small" color={9}>
                Have a license key? Paste it here.
              </Text.Text>
              <KeyInput />
              {result.variant === "error" && (
                <Status.Summary level="small" status={result.status} />
              )}
            </Flex.Box>
          </Flex.Box>
        </Flex.Box>
        <Flex.Box x align="center" justify="center" gap="small">
          {!Session.Runtime.CORE_EMBEDDED && (
            <Button.Button variant="text" size="small" textColor={9} onClick={logout}>
              Log out
            </Button.Button>
          )}
          {onBack != null && (
            <Button.Button variant="text" size="small" textColor={9} onClick={onBack}>
              Back
            </Button.Button>
          )}
        </Flex.Box>
      </Flex.Box>
    </Shell.Frame>
  );
};
