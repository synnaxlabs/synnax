// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/vcs/useConnectModal.css";

import { type vcs } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useState } from "react";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";
import { Vcs } from "@/platform/vcs";

const repoName = (url: string): string =>
  url.replace(/^[a-z]+:\/\//, "").replace(/\.git$/, "");

const Connect = ({ close }: Modals.ContentProps<Record<never, never>, void>) => {
  const client = Synnax.use();
  const { status } = Vcs.useStatus();
  const setStatus = Vcs.useSetStatus();
  const addStatus = Status.useAdder();
  const started = status?.enabled === true;
  const [configuration, setConfiguration] = useState(
    started ? status.configuration : "main",
  );
  const [remoteURL, setRemoteURL] = useState(started ? status.remoteUrl : "");
  const [token, setToken] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const tokenSet = status?.tokenSet === true;
  const handleConnect = async () => {
    if (client == null) return;
    setLoading(true);
    setError(null);
    let next: vcs.Status | null = null;
    try {
      next = await client.vcs.connect({
        configuration: configuration.trim(),
        remoteUrl: remoteURL.trim(),
        token,
      });
    } catch (e) {
      setError(errors.fromUnknown(e).message);
    }
    setLoading(false);
    if (next == null) return;
    setStatus(next);
    addStatus({
      variant: "success",
      name: next.configuration,
      message: started ? "Saved connection" : "Started version control",
      description:
        next.remoteUrl.length > 0
          ? `New commits go to ${repoName(next.remoteUrl)}.`
          : "Commits stay on this Core until you add a Git URL.",
    });
    close();
  };
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.SourceControl />}>
        {started ? "Connection settings" : "Start version control"}
      </Modal.Header>
      <Modal.Body gap="large" align="stretch">
        <Text.Text level="small" color={9} className={CSS.BE("vcs-connect", "intro")}>
          The Core keeps its tasks, Arc programs, schematics, plots, and channels in a
          Git repository. Without a Git URL, commits stay on this Core.
        </Text.Text>
        <Flex.Box x align="start" gap="medium">
          <Input.Item
            label="Configuration"
            required
            padHelpText={false}
            className={CSS.BE("vcs-connect", "configuration")}
          >
            <Input.Text
              autoFocus={!started}
              placeholder="main"
              value={configuration}
              onChange={setConfiguration}
              disabled={started}
            />
          </Input.Item>
          <Input.Item label="Git URL" padHelpText={false} grow>
            <Input.Text
              autoFocus={started}
              placeholder="https://github.com/owner/repository.git"
              value={remoteURL}
              onChange={setRemoteURL}
            />
          </Input.Item>
        </Flex.Box>
        <Input.Item
          label="Access token"
          padHelpText={false}
          helpText={
            tokenSet
              ? "A token is saved. Leave this empty to keep it."
              : "Needs read and write access to the repository. The Core never sends it back."
          }
        >
          <Input.Text
            type="password"
            placeholder={tokenSet ? "Saved" : "Personal access token"}
            value={token}
            onChange={setToken}
          />
        </Input.Item>
        {error != null && (
          <Status.Summary
            variant="error"
            center={false}
            message={started ? "Failed to save connection" : "Failed to start"}
            description={error}
          />
        )}
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End x align="center">
          <Button.Button
            variant="filled"
            status={loading ? "loading" : undefined}
            disabled={client == null || configuration.trim().length === 0}
            onClick={() => void handleConnect()}
            trigger={Triggers.SAVE}
            triggerIndicator
          >
            {started ? "Save" : "Start version control"}
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/** Opens the dialog that starts version control or changes its connection. */
export const useConnectModal = Modals.create(Connect);
