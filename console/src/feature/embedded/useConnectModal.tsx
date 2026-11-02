// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/embedded/useConnectModal.css";

import { type Synnax as Client } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Description } from "@synnaxlabs/lyra/description";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Status } from "@synnaxlabs/lyra/status";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { countSessions, ensure, replace } from "@/feature/embedded/apiKey";
import { useStatus } from "@/feature/embedded/Provider";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";

const CLIENT_DOCS_URL = "https://docs.synnaxlabs.com/reference/client";

const MASK = "•".repeat(16);

const PYTHON_CODE = `import synnax as sy

client = sy.Synnax()`;

/**
 * @returns The client release that matches the given Core version, or null for a
 * development build, which no release matches.
 */
const matchingRelease = (version: string | undefined): string | null => {
  const [major, minor] = version?.split(".") ?? [];
  if (major == null || minor == null || (major === "0" && minor === "0")) return null;
  return `${major}.${minor}.0`;
};

const typeScriptCode = (host: string, port: number, apiKey: string): string =>
  `import { Synnax } from "@synnaxlabs/client";

const client = new Synnax({
  host: "${host}",
  port: ${port},
  apiKey: "${apiKey}",
});`;

interface SnippetProps {
  /** What the snippet is, as the copy button names it. */
  label: string;
  code: string;
  /** The text to copy, when it differs from what the snippet shows. */
  copy?: string;
}

const Snippet = ({ label, code, copy = code }: SnippetProps): ReactElement => (
  <Flex.Box x align="start" className={CSS.BE("connect", "snippet")}>
    <pre aria-label={label}>{code}</pre>
    <Button.Copy
      text={copy}
      variant="text"
      size="small"
      tooltip={`Copy ${label.toLowerCase()}`}
      textColor={9}
    />
  </Flex.Box>
);

interface Params {
  host: string;
  port: number;
  apiKey: string;
}

interface TabProps {
  /** The client release that matches the Core, or null for any release. */
  release: string | null;
}

const PythonTab = ({ release }: TabProps): ReactElement => (
  <Flex.Box y gap="medium">
    <Text.Text color={9}>
      Install the client, then connect. The client finds Synnax Desktop by itself.
    </Text.Text>
    <Snippet
      label="Install command"
      code={release == null ? "pip install synnax" : `pip install "synnax~=${release}"`}
    />
    <Snippet label="Python code" code={PYTHON_CODE} />
  </Flex.Box>
);

interface TypeScriptTabProps extends TabProps, Params {}

const TypeScriptTab = ({
  release,
  host,
  port,
  apiKey,
}: TypeScriptTabProps): ReactElement => (
  <Flex.Box y gap="medium">
    <Text.Text color={9}>
      Install the client, then connect with this code. It holds the API key, so keep it
      private.
    </Text.Text>
    <Snippet
      label="Install command"
      code={
        release == null
          ? "npm install @synnaxlabs/client"
          : `npm install @synnaxlabs/client@~${release}`
      }
    />
    <Snippet
      label="TypeScript code"
      code={typeScriptCode(host, port, `${apiKey.slice(0, 4)}${MASK}`)}
      copy={typeScriptCode(host, port, apiKey)}
    />
  </Flex.Box>
);

interface ValueProps {
  label: string;
  value: string;
  /** What the value shows in place of the real one. */
  display?: string;
}

const Value = ({ label, value, display = value }: ValueProps): ReactElement => (
  <Description.Item>
    <Description.Label>{label}</Description.Label>
    <Description.Value>
      <Flex.Box x align="center" gap="small">
        <span className={CSS.BE("connect", "value")} aria-label={label}>
          {display}
        </span>
        <Button.Copy
          text={value}
          variant="text"
          size="small"
          tooltip={`Copy ${label.toLowerCase()}`}
          textColor={9}
        />
      </Flex.Box>
    </Description.Value>
  </Description.Item>
);

const OtherTab = ({ host, port, apiKey }: Params): ReactElement => {
  const [revealed, setRevealed] = useState(false);
  return (
    <Flex.Box y gap="medium">
      <Text.Text color={9}>
        Give these values to the C++ client, LabVIEW, or any other client.
      </Text.Text>
      <Description.List>
        <Value label="Host" value={host} />
        <Value label="Port" value={port.toString()} />
        <Value
          label="API key"
          value={apiKey}
          display={revealed ? apiKey : `${apiKey.slice(0, 4)}${MASK}`}
        />
      </Description.List>
      <Button.Button
        variant="text"
        size="small"
        onClick={() => setRevealed((r) => !r)}
        className={CSS.BE("connect", "reveal")}
      >
        {revealed ? <Icon.Hidden /> : <Icon.Visible />}
        {revealed ? "Hide API key" : "Show API key"}
      </Button.Button>
    </Flex.Box>
  );
};

interface KeyState {
  apiKey: string;
  sessions: number;
}

const retrieveKeyState = async (client: Client): Promise<KeyState> => {
  const apiKey = await ensure(client);
  return { apiKey, sessions: await countSessions(client, apiKey) };
};

const formatSessions = (count: number): string =>
  `${count} ${count === 1 ? "session" : "sessions"}`;

const ConnectModal = (): ReactElement => {
  const client = Synnax.use();
  const status = useStatus();
  const connectionStatus = Synnax.useConnectionStatus();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const confirm = Modals.useConfirm();
  const [state, setState] = useState<KeyState | null>(null);
  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      try {
        const next = await retrieveKeyState(client);
        if (!signal.aborted) setState(next);
      } catch (err) {
        if (!signal.aborted) handleError(err, "Failed to load API key");
      }
    },
    [client, handleError],
  );
  const handleReset = (): void =>
    handleError(async () => {
      if (client == null) return;
      const confirmed = await confirm({
        message: "Are you sure you want to reset the API key?",
        description:
          "Every script that uses the current key disconnects. Python scripts that call sy.Synnax() use the new key the next time they start. Code that holds the key needs the new one.",
        confirm: { label: "Reset key" },
      });
      if (confirmed !== true) return;
      const apiKey = await replace(client);
      setState({ apiKey, sessions: await countSessions(client, apiKey) });
      addStatus({ variant: "success", message: "Reset API key" });
    }, "Failed to reset API key");
  const connection = status.state === "running" ? status.connection : null;
  const release = matchingRelease(connectionStatus.details.nodeVersion);
  let body: ReactElement;
  if (connection == null || state == null)
    body = (
      <Status.Summary
        variant="loading"
        message={connection == null ? "Waiting for Synnax to start..." : "Loading..."}
      />
    );
  else {
    const { host, port } = connection;
    const { apiKey } = state;
    body = (
      <Tabs.Frame initialValue="python" empty={false} gap="large">
        <Tabs.Selector variant="chip">
          <Tabs.Tab itemKey="python">Python</Tabs.Tab>
          <Tabs.Tab itemKey="typescript">TypeScript</Tabs.Tab>
          <Tabs.Tab itemKey="other">Other</Tabs.Tab>
        </Tabs.Selector>
        <Tabs.Content itemKey="python">
          <PythonTab release={release} />
        </Tabs.Content>
        <Tabs.Content itemKey="typescript">
          <TypeScriptTab release={release} host={host} port={port} apiKey={apiKey} />
        </Tabs.Content>
        <Tabs.Content itemKey="other">
          <OtherTab host={host} port={port} apiKey={apiKey} />
        </Tabs.Content>
      </Tabs.Frame>
    );
  }
  return (
    <Modal.Frame className={CSS.B("connect")}>
      <Modal.Header icon={<Icon.Code />}>Connect from code</Modal.Header>
      <Modal.Body className={CSS.BE("connect", "body")} gap="large">
        {body}
      </Modal.Body>
      <Modal.Footer>
        <Flex.Box x gap="small" align="center" className={CSS.BE("connect", "actions")}>
          {state != null && (
            <Text.Text
              level="small"
              color={9}
              className={CSS.BE("connect", "sessions")}
            >
              {formatSessions(state.sessions)}
            </Text.Text>
          )}
          <Button.Button variant="text" href={CLIENT_DOCS_URL} target="_blank">
            <Icon.OpenExternal />
            Client docs
          </Button.Button>
          <Button.Button
            variant="outlined"
            status="error"
            disabled={state == null}
            onClick={handleReset}
          >
            <Icon.Refresh />
            Reset key
          </Button.Button>
        </Flex.Box>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/**
 * Opens a dialog that shows how scripts on this computer connect to Synnax Desktop:
 * install commands and code for each client, the address and API key, and the number
 * of sessions the key has open. From it, a person can replace the key.
 */
export const useConnectModal = Modals.create(ConnectModal);
