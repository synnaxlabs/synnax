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
import { Button, Flex, Icon, Input, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Shell } from "@/feature/shell";
import { Clipboard } from "@/platform/clipboard";
import { Connection } from "@/platform/connection";
import { CSS } from "@/platform/css";
import { License } from "@/platform/license";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

/** The extension the hub gives a downloaded token file. */
const TOKEN_FILE_EXTENSION = "license";

const decoder = new TextDecoder();

interface FingerprintProps {
  info: License.InfoResult;
}

const Fingerprint = ({ info: { info, error } }: FingerprintProps): ReactElement => {
  const copy = Clipboard.useCopy();
  const fingerprint = info?.fingerprint ?? [];
  const copyFingerprint = (): void =>
    copy(License.joinFingerprint(fingerprint), "host fingerprint");
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
            <Text.Text key={hash} variant="code" level="small" overflow="ellipsis">
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
        href={License.ACTIVATE_URL}
        target="_blank"
        className={CSS.BE("license-activate", "account")}
      >
        <Icon.OpenExternal />
        Get a token from your account
      </Button.Button>
    </Flex.Box>
  );
};

export interface ActivateProps {
  /**
   * True when the session has one fixed Core. The screen then shows no connection
   * island and no log out action.
   */
  standalone?: boolean;
  /** Offers a way back to the screen that led here. */
  onBack?: () => void;
}

/**
 * Full-screen activation surface for a Core that refuses requests until a license
 * applies. Shows the host fingerprint the hub needs and takes the token it issues,
 * pasted or from a file.
 */
export const Activate = ({
  standalone = false,
  onBack,
}: ActivateProps): ReactElement => {
  const client = Synnax.use();
  const connection = Synnax.useConnectionStatus();
  const target = Session.Core.useSelectSelected();
  const logout = Session.useLogout();
  const handleError = Status.useErrorHandler();
  const info = License.useInfo();
  const [token, setToken] = useState("");
  const [activating, setActivating] = useState(false);
  const [result, setResult] = useState<status.Status | null>(null);

  const pickFile = (): void =>
    handleError(async () => {
      const file = await Runtime.pickFiles({
        title: "Select a license file",
        extension: TOKEN_FILE_EXTENSION,
      });
      if (file == null) return;
      const bytes = await Runtime.toBytes(await file.read());
      setToken(decoder.decode(bytes).trim());
    }, "Failed to read the license file");

  const activate = (): void => {
    const trimmed = token.trim();
    if (client == null || trimmed === "") return;
    setActivating(true);
    setResult(null);
    void (async () => {
      try {
        await client.license.activate(trimmed);
        setResult(status.create({ variant: "success", message: "License activated" }));
      } catch (error) {
        setResult(status.fromException(error, "Failed to activate the license"));
      } finally {
        setActivating(false);
      }
    })();
  };

  return (
    <Shell.Frame
      className={CSS.B("license-activate")}
      connection={standalone ? null : target}
    >
      <Flex.Box y gap="large" className={CSS.BE("license-activate", "body")}>
        <Status.Summary variant="warning" level="h4" message={connection.message} />
        <Fingerprint info={info} />
        <Flex.Box y gap="small" full="x">
          <Text.Text level="small" weight={500} color={10}>
            License token
          </Text.Text>
          <Input.Text
            area
            value={token}
            onChange={setToken}
            placeholder="Paste the token"
            className={CSS.BE("license-activate", "token")}
          />
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
              onClick={activate}
              disabled={activating || token.trim() === ""}
            >
              Activate
            </Button.Button>
          </Flex.Box>
          {result != null && <Status.Summary status={result} level="small" />}
        </Flex.Box>
        <Flex.Box x gap="small" className={CSS.BE("license-activate", "actions")}>
          <Connection.Retry variant="outlined" grow justify="center">
            Check again
          </Connection.Retry>
          {!standalone && (
            <Button.Button variant="outlined" grow justify="center" onClick={logout}>
              <Icon.Logout />
              Log out
            </Button.Button>
          )}
          {onBack != null && (
            <Button.Button variant="outlined" grow justify="center" onClick={onBack}>
              <Icon.Arrow.Left />
              Back
            </Button.Button>
          )}
        </Flex.Box>
      </Flex.Box>
    </Shell.Frame>
  );
};
