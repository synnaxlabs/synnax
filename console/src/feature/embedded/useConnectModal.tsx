// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/embedded/useConnectModal.css";

import { type resource } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Status } from "@synnaxlabs/lyra/status";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { Credential, Synnax } from "@synnaxlabs/pluto";
import { Fragment, type PropsWithChildren, type ReactElement, useState } from "react";

import { countSessions, ensure, replace } from "@/feature/embedded/apiKey";
import { useStatus } from "@/feature/embedded/Provider";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";

const CLIENT_DOCS_URL = "https://docs.synnaxlabs.com/reference/client";

const MASK = "•".repeat(16);

/** A run of code, colored by its kind when it has one. */
type Token =
  string | { kind: "keyword" | "string" | "function" | "comment"; text: string };

/** Lines of tokens. One source gives both the colored code and the copied text. */
type Code = Token[][];

const keyword = (text: string): Token => ({ kind: "keyword", text });
const quoted = (text: string): Token => ({ kind: "string", text: `"${text}"` });
const call = (text: string): Token => ({ kind: "function", text });
const comment = (text: string): Token => ({ kind: "comment", text });

const toText = (code: Code): string =>
  code
    .map((line) => line.map((t) => (typeof t === "string" ? t : t.text)).join(""))
    .join("\n");

const PYTHON_CODE: Code = [
  [keyword("import"), " synnax ", keyword("as"), " sy"],
  [],
  [comment("# Finds Synnax Desktop on this computer.")],
  ["client = sy.", call("Synnax"), "()"],
];

/**
 * @returns The client release that matches the given Core version, or null for a
 * development build, which no release matches.
 */
const matchingRelease = (version: string | undefined): string | null => {
  const [major, minor] = version?.split(".") ?? [];
  if (major == null || minor == null || (major === "0" && minor === "0")) return null;
  return `${major}.${minor}.0`;
};

const pythonInstall = (release: string | null): Code => [
  [call("pip"), " install ", release == null ? "synnax" : quoted(`synnax~=${release}`)],
];

const typeScriptInstall = (release: string | null): Code => [
  [call("npm"), " install @synnaxlabs/client", release == null ? "" : `@~${release}`],
];

const typeScriptCode = (host: string, port: number, apiKey: string): Code => [
  [
    keyword("import"),
    " { Synnax } ",
    keyword("from"),
    " ",
    quoted("@synnaxlabs/client"),
    ";",
  ],
  [],
  [keyword("const"), " client = ", keyword("new"), " ", call("Synnax"), "({"],
  ["  host: ", quoted(host), ","],
  [`  port: ${port},`],
  ["  apiKey: ", quoted(apiKey), ","],
  ["});"],
];

interface SnippetProps {
  /** What the snippet holds, as assistive technology names it. */
  label: string;
  code: Code;
  /** The code to copy, when it differs from what the snippet shows. */
  copy?: Code;
}

const Snippet = ({ label, code, copy = code }: SnippetProps): ReactElement => (
  <Flex.Box x align="start" className={CSS.BE("connect", "snippet")}>
    <pre aria-label={label}>
      {code.map((line, i) => (
        <Fragment key={i}>
          {i > 0 && "\n"}
          {line.map((token, j) =>
            typeof token === "string" ? (
              token
            ) : (
              <span key={j} className={CSS.BEM("connect", "token", token.kind)}>
                {token.text}
              </span>
            ),
          )}
        </Fragment>
      ))}
    </pre>
    <Button.Copy
      text={toText(copy)}
      variant="text"
      size="small"
      textColor={9}
      aria-label={`Copy ${label}`}
    />
  </Flex.Box>
);

interface StepProps extends PropsWithChildren {
  title: string;
}

const Step = ({ title, children }: StepProps): ReactElement => (
  <Flex.Box y gap="small">
    <Text.Text level="small" color={9}>
      {title}
    </Text.Text>
    {children}
  </Flex.Box>
);

interface TabProps {
  /** The client release that matches the Core, or null for any release. */
  release: string | null;
}

const PythonTab = ({ release }: TabProps): ReactElement => (
  <Flex.Box y gap="large">
    <Step title="Install">
      <Snippet label="Python install command" code={pythonInstall(release)} />
    </Step>
    <Step title="Connect">
      <Snippet label="Python code" code={PYTHON_CODE} />
    </Step>
  </Flex.Box>
);

interface TypeScriptTabProps extends TabProps {
  host: string;
  port: number;
  apiKey: string;
}

const TypeScriptTab = ({
  release,
  host,
  port,
  apiKey,
}: TypeScriptTabProps): ReactElement => (
  <Flex.Box y gap="large">
    <Step title="Install">
      <Snippet label="TypeScript install command" code={typeScriptInstall(release)} />
    </Step>
    <Step title="Connect">
      <Snippet
        label="TypeScript code"
        code={typeScriptCode(host, port, `${apiKey.slice(0, 4)}${MASK}`)}
        copy={typeScriptCode(host, port, apiKey)}
      />
      <Text.Text level="small" color={9}>
        The code holds the API key, so keep it private.
      </Text.Text>
    </Step>
  </Flex.Box>
);

const formatSessions = (count: number): string =>
  `${count} ${count === 1 ? "session" : "sessions"}`;

interface SessionCountProps {
  apiKey: string;
  /** The user that owns the key. */
  subject: resource.ID;
}

const SessionCount = ({ apiKey, subject }: SessionCountProps): ReactElement | null => {
  const sessions = Credential.useResultSessions({ subject });
  if (sessions.variant !== "success" && sessions.variant !== "error") return null;
  return (
    <Text.Text color={9}>
      {sessions.variant === "success"
        ? formatSessions(countSessions(sessions.data, apiKey))
        : "Sessions unknown"}
    </Text.Text>
  );
};

const ConnectModal = (): ReactElement => {
  const client = Synnax.use();
  const status = useStatus();
  const connectionStatus = Synnax.useConnectionStatus();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const confirm = Modals.useConfirm();
  const invalidateSessions = Credential.useInvalidateSessions();
  const [apiKey, setAPIKey] = useState<string | null>(null);
  const subject = client?.auth.subject;
  // Sessions have no live updates: drop the cached answer before a new count mounts.
  const show = (next: string): void => {
    if (subject != null) invalidateSessions({ subject });
    setAPIKey(next);
  };
  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      try {
        const next = await ensure(client);
        if (!signal.aborted) show(next);
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
      show(await replace(client));
      addStatus({ variant: "success", message: "Reset API key" });
    }, "Failed to reset API key");
  const connection = status.state === "running" ? status.connection : null;
  const release = matchingRelease(connectionStatus.details.nodeVersion);
  let body: ReactElement;
  if (connection == null || apiKey == null)
    body = (
      <Status.Summary
        variant="loading"
        message={connection == null ? "Waiting for Synnax to start..." : "Loading..."}
      />
    );
  else {
    const { host, port } = connection;
    body = (
      <Tabs.Frame initialValue="python" empty={false} gap="large">
        <Tabs.Selector variant="chip">
          <Tabs.Tab itemKey="python">
            <Icon.Python />
            Python
          </Tabs.Tab>
          <Tabs.Tab itemKey="typescript">
            <Icon.TypeScript />
            TypeScript
          </Tabs.Tab>
        </Tabs.Selector>
        <Tabs.Content itemKey="python">
          <PythonTab release={release} />
        </Tabs.Content>
        <Tabs.Content itemKey="typescript">
          <TypeScriptTab release={release} host={host} port={port} apiKey={apiKey} />
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
        <Nav.Bar.Start>
          {apiKey != null && subject != null && (
            <SessionCount key={apiKey} apiKey={apiKey} subject={subject} />
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End align="center" gap="small">
          <Button.Button variant="text" href={CLIENT_DOCS_URL} target="_blank">
            <Icon.OpenExternal />
            Client docs
          </Button.Button>
          <Button.Button
            variant="outlined"
            status="error"
            disabled={apiKey == null}
            onClick={handleReset}
          >
            <Icon.Refresh />
            Reset key
          </Button.Button>
        </Nav.Bar.End>
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
