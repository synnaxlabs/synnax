// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/license/Activate.css";

import { ExpiredLicenseError, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type Triggers } from "@synnaxlabs/lyra/triggers";
import { License as PLicense, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Shell } from "@/feature/shell";
import { CSS } from "@/platform/css";
import { License } from "@/platform/license";
import { Runtime } from "@/platform/runtime";
import { Shell as PlatformShell } from "@/platform/shell";
import { Session } from "@/session";

/** The portal page that issues a license key for a host fingerprint. */
const PORTAL_ACTIVATE_URL = "https://portal.synnaxlabs.com/licenses/activate";

const KEY_FILE_EXTENSION = "lic";

const ACTIVATE_TRIGGER: Triggers.Trigger = ["Enter"];

const decoder = new TextDecoder();

const portalURL = (fingerprint: string[]): string => {
  if (fingerprint.length === 0) return PORTAL_ACTIVATE_URL;
  const params = new URLSearchParams({
    fingerprint: License.joinFingerprint(fingerprint),
  });
  return `${PORTAL_ACTIVATE_URL}?${params.toString()}`;
};

/**
 * Full-screen activation surface for a Core that refuses requests until a license
 * applies. The connection check keeps polling, so the screen leaves on its own once a
 * license applies from anywhere.
 */
export const Activate = (): ReactElement => {
  const client = Synnax.use();
  const { details } = Synnax.useConnectionStatus();
  const target = Session.Core.useSelectSelected();
  const logout = Session.useLogout();
  const handleError = Status.useErrorHandler();
  const result = PLicense.useResult({});
  const [key, setKey] = useState("");
  const [activating, setActivating] = useState(false);
  const [error, setError] = useState<status.Status | null>(null);
  const fingerprint = result.data?.fingerprint ?? [];
  const expired = ExpiredLicenseError.matches(details.error);
  const noHardware = result.data != null && fingerprint.length === 0;

  const pickFile = (): void =>
    handleError(async () => {
      const file = await Runtime.pickFiles({
        title: "Select a license file",
        extension: KEY_FILE_EXTENSION,
      });
      if (file == null) return;
      const bytes = await Runtime.toBytes(await file.read());
      setKey(decoder.decode(bytes).trim());
    }, "Failed to read the license file");

  const activate = async (): Promise<void> => {
    if (client == null || key.trim() === "") return;
    setActivating(true);
    setError(null);
    try {
      await client.license.activate(key.trim());
    } catch (e) {
      setError(status.fromException(e, "Failed to activate the license"));
    } finally {
      setActivating(false);
    }
  };

  let description = "Get a key from your Synnax account, then paste it here.";
  if (noHardware)
    description = "This Core reports no network hardware. Ask for a floating license.";
  else if (expired) description = "Get a new key from your Synnax account.";

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
          <Flex.Box y gap="small" full="x">
            <Input.Text
              size="large"
              value={key}
              onChange={setKey}
              placeholder="Paste a license key"
              aria-label="License key"
              full="x"
              disabled={activating}
            >
              <Button.Button
                size="large"
                variant="outlined"
                onClick={pickFile}
                disabled={activating}
                tooltip="Load a license file"
                aria-label="Load a license file"
              >
                <Icon.Attachment />
              </Button.Button>
            </Input.Text>
            <Button.Button
              variant="filled"
              size="large"
              full="x"
              justify="center"
              trigger={ACTIVATE_TRIGGER}
              onClick={() => void activate()}
              status={activating ? "loading" : undefined}
              disabled={key.trim() === ""}
            >
              Activate
              <Icon.Arrow.Right />
            </Button.Button>
            <Flex.Box className={CSS.BE("license-activate", "status")}>
              {error != null && <Status.Summary status={error} level="small" />}
              {error == null && result.variant === "error" && (
                <Status.Summary level="small" status={result.status} />
              )}
            </Flex.Box>
          </Flex.Box>
        </Flex.Box>
        <Flex.Box x align="center" justify="center" gap="small">
          <Button.Button
            variant="text"
            size="small"
            textColor={9}
            href={portalURL(fingerprint)}
            target="_blank"
          >
            Get a key
            <Icon.OpenExternal />
          </Button.Button>
          <Button.Copy
            variant="text"
            size="small"
            textColor={9}
            disabled={fingerprint.length === 0}
            text={() => License.joinFingerprint(fingerprint)}
          >
            Copy fingerprint
          </Button.Copy>
          {!Session.Runtime.CORE_EMBEDDED && (
            <Button.Button variant="text" size="small" textColor={9} onClick={logout}>
              Log out
            </Button.Button>
          )}
        </Flex.Box>
      </Flex.Box>
    </Shell.Frame>
  );
};
