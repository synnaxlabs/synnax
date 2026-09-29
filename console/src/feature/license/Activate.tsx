// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/license/Activate.css";

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Shell } from "@/feature/shell";
import { Clipboard } from "@/platform/clipboard";
import { Connection } from "@/platform/connection";
import { CSS } from "@/platform/css";
import { License } from "@/platform/license";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

/** The portal page that issues a license key for a host fingerprint. */
const PORTAL_ACTIVATE_URL = "https://docs.synnaxlabs.com/licenses/activate";

const KEY_FILE_EXTENSION = "license";

const decoder = new TextDecoder();

interface FingerprintProps {
  info: License.InfoResult;
}

const Fingerprint = ({ info: { info, error } }: FingerprintProps): ReactElement => {
  const copy = Clipboard.useCopy();
  const fingerprint = info?.fingerprint ?? [];
  const copyFingerprint = (): void => copy(fingerprint.join(", "), "host fingerprint");
  return (
    <Flex.Box y gap="small" full="x">
      <Flex.Box x align="center" justify="between">
        <Text.Text level="small" weight={500} color={10}>
          Host fingerprint
        </Text.Text>
        <Button.Button
          variant="text"
          size="small"
          disabled={fingerprint.length === 0}
          onClick={copyFingerprint}
        >
          <Icon.Copy />
          Copy
        </Button.Button>
      </Flex.Box>
      {fingerprint.length > 0 && (
        <Flex.Box y gap="tiny" className={CSS.BE("license-activate", "hashes")}>
          {fingerprint.map((hash) => (
            <Text.Text
              key={hash}
              variant="code"
              level="small"
              overflow="ellipsis"
              className={CSS.BE("license-activate", "hash")}
            >
              {hash}
            </Text.Text>
          ))}
        </Flex.Box>
      )}
      {info != null && fingerprint.length === 0 && (
        <Text.Text level="small" color={9}>
          This Core reports no network hardware. Ask for a floating license.
        </Text.Text>
      )}
      {error != null && (
        <Status.Summary
          variant="error"
          level="small"
          message="Failed to read the host fingerprint"
          description={error.message}
        />
      )}
      <Button.Button
        variant="text"
        size="small"
        href={PORTAL_ACTIVATE_URL}
        target="_blank"
        className={CSS.BE("license-activate", "portal")}
      >
        <Icon.OpenExternal />
        Get a license key from the portal
      </Button.Button>
    </Flex.Box>
  );
};

/**
 * Full-screen activation surface for a Core that refuses requests until a license
 * applies.
 */
export const Activate = (): ReactElement => {
  const client = Synnax.use();
  const connection = Synnax.useConnectionStatus();
  const target = Session.Core.useSelectSelected();
  const logout = Session.useLogout();
  const handleError = Status.useErrorHandler();
  const info = License.useInfo();
  const [key, setKey] = useState("");
  const [activating, setActivating] = useState(false);
  const [error, setError] = useState<status.Status | null>(null);

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
    if (client == null) return;
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

  return (
    <Shell.Frame
      className={CSS.B("license-activate")}
      connection={Session.Runtime.CORE_EMBEDDED ? null : target}
    >
      <Flex.Box y gap="large" className={CSS.BE("license-activate", "body")}>
        <Status.Summary
          variant="warning"
          level="h4"
          message={connection.message}
          description={info.info?.warning}
        />
        <Fingerprint info={info} />
        <Flex.Box y gap="small" full="x">
          <Input.Item label="License key">
            <Input.Text
              area
              value={key}
              onChange={setKey}
              placeholder="Paste the license key"
              className={CSS.BE("license-activate", "key")}
            />
          </Input.Item>
          <Flex.Box x gap="small" className={CSS.BE("license-activate", "actions")}>
            <Button.Button
              variant="outlined"
              grow
              justify="center"
              onClick={pickFile}
              disabled={activating}
            >
              <Icon.Attachment />
              Select file
            </Button.Button>
            <Button.Button
              variant="filled"
              grow
              justify="center"
              onClick={() => void activate()}
              disabled={activating || key.trim() === ""}
            >
              Activate
            </Button.Button>
          </Flex.Box>
          {error != null && <Status.Summary status={error} level="small" />}
        </Flex.Box>
        <Flex.Box x gap="small" className={CSS.BE("license-activate", "actions")}>
          <Connection.Retry variant="outlined" grow justify="center">
            Check again
          </Connection.Retry>
          {!Session.Runtime.CORE_EMBEDDED && (
            <Button.Button variant="outlined" grow justify="center" onClick={logout}>
              <Icon.Logout />
              Log out
            </Button.Button>
          )}
        </Flex.Box>
      </Flex.Box>
    </Shell.Frame>
  );
};
