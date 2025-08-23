// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Input, Nav, Text } from "@synnaxlabs/pluto";
import { useState } from "react";

import { testConnection } from "@/cluster/testConnection";
import { type BaseArgs, createBase, type Prompt } from "@/modals/Base";
import { ModalContentLayout } from "@/modals/layout";
import { Triggers } from "@/triggers";

export interface Credentials {
  username: string;
  password: string;
}

export interface PromptCredentialsLayoutArgs extends BaseArgs<Credentials> {
  host?: string;
}

export const CREDENTIALS_LAYOUT_TYPE = "credentials";

export interface PromptCredentials
  extends Prompt<Credentials, PromptCredentialsLayoutArgs> {}

export const [useCredentials, Credentials] = createBase<
  Credentials,
  PromptCredentialsLayoutArgs
>(
  "Cluster Credentials",
  CREDENTIALS_LAYOUT_TYPE,
  ({ value: { result, host }, onFinish }) => {
    const [username, setUsername] = useState(result?.username ?? "");
    const [password, setPassword] = useState(result?.password ?? "");
    const [error, setError] = useState<string | undefined>(undefined);
    const [testing, setTesting] = useState(false);

    const handleSubmit = async () => {
      if (username.length === 0) return setError("Username is required");
      if (password.length === 0) return setError("Password is required");

      setError(undefined);
      setTesting(true);

      try {
        const [hostName, portStr] = (host ?? "localhost:9090").split(":");
        const port = parseInt(portStr) || 9090;

        const state = await testConnection({
          host: hostName,
          port,
          username,
          password,
          secure: false,
        });

        if (state.status !== "connected") {
          setError("Invalid credentials or connection failed");
          return;
        }

        onFinish({ username, password });
      } catch (err) {
        setError(
          "Connection failed: " +
            (err instanceof Error ? err.message : "Unknown error"),
        );
      } finally {
        setTesting(false);
      }
    };

    const footer = (
      <>
        <Triggers.SaveHelpText action="Connect" trigger={Triggers.SAVE} />
        <Nav.Bar.End x align="center">
          <Button.Button
            status="success"
            disabled={username.length === 0 || password.length === 0 || testing}
            variant="filled"
            onClick={handleSubmit}
            trigger={Triggers.SAVE}
          >
            {testing ? "Testing..." : "Connect"}
          </Button.Button>
        </Nav.Bar.End>
      </>
    );

    return (
      <ModalContentLayout footer={footer}>
        {host && (
          <Text.Text level="p" style={{ marginBottom: "1rem", opacity: 0.7 }}>
            Connecting to: {host}
          </Text.Text>
        )}
        <Flex.Box x>
          <Input.Item
            label="Username"
            required
            helpText={error}
            status={error != null ? "error" : "success"}
            padHelpText={false}
          >
            <Input.Text
              autoFocus
              placeholder="Username"
              value={username}
              onChange={setUsername}
            />
          </Input.Item>
          <Input.Item label="Password" required padHelpText>
            <Input.Text
              type="password"
              placeholder="Password"
              value={password}
              onChange={setPassword}
            />
          </Input.Item>
        </Flex.Box>
        {error && (
          <Text.Text
            level="p"
            style={{ marginTop: "0.5rem", color: "var(--pluto-error-z)" }}
          >
            {error}
          </Text.Text>
        )}
      </ModalContentLayout>
    );
  },
);
