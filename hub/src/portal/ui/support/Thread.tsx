// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Icon, Input, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";

import { post, reload } from "@/portal/ui/api";
import { dateTime } from "@/portal/ui/format";
import { Page } from "@/portal/ui/Page";
import { ThreadStatus } from "@/portal/ui/support/ThreadStatus";
import { useAction } from "@/portal/ui/useAction";
import {
  type Message,
  type Organization,
  type Sender,
  type Thread as ThreadRecord,
} from "@/server/db/schema";
import { type Status } from "@/server/support/tracker";

export interface ThreadProps {
  thread: ThreadRecord;
  status: Status;
  messages: Message[];
  organization: Organization | null;
  /** sender is how the viewer's replies are attributed. */
  sender: Sender;
  staff: boolean;
}

const URL_PATTERN = /(https?:\/\/[^\s<]+[^\s<.,;:!?)\]])/g;

/** linkify splits plain text so that URLs render as links. */
const linkify = (text: string): ReactElement[] =>
  text.split(URL_PATTERN).map((part, i) =>
    URL_PATTERN.test(part) ? (
      <Text.Text
        key={i}
        el="a"
        variant="link"
        href={part}
        target="_blank"
        rel="noreferrer"
      >
        {part}
      </Text.Text>
    ) : (
      <span key={i}>{part}</span>
    ),
  );

/** Thread shows one support conversation and a composer to reply. */
export const Thread = ({
  thread,
  status,
  messages,
  organization,
  sender,
  staff,
}: ThreadProps): ReactElement => (
  <Page
    title={thread.title}
    subtitle={
      <Flex.Box x align="center" gap="small" wrap>
        <ThreadStatus status={status} />
        {organization?.kind === "team" && (
          <Text.Text level="p" color={9}>
            {organization.name}
          </Text.Text>
        )}
        {staff && (
          <Text.Text level="p" variant="link" href={thread.issueURL} target="_blank">
            {thread.issueIdentifier}
          </Text.Text>
        )}
      </Flex.Box>
    }
    actions={
      <Button.Button variant="text" href="/portal/support">
        <Icon.Caret.Left />
        All threads
      </Button.Button>
    }
  >
    <Flex.Box y gap="medium" full="x">
      {messages.map((m) => (
        <Flex.Box
          key={m.key}
          y
          gap="small"
          bordered
          rounded
          background={m.sender === "staff" ? 1 : 2}
          className={`portal-message portal-message--${m.sender}`}
          style={{ padding: "2.5rem 3rem" }}
        >
          <Flex.Box x justify="between" align="center" gap="medium">
            <Text.Text level="small" weight={500}>
              {m.sender === "staff" ? `${m.author} (Synnax Labs)` : m.author}
            </Text.Text>
            <Text.Text level="small" color={9}>
              {dateTime(m.at)}
            </Text.Text>
          </Flex.Box>
          <Text.Text level="p">{linkify(m.text)}</Text.Text>
        </Flex.Box>
      ))}
    </Flex.Box>
    <Composer threadKey={thread.key} sender={sender} />
  </Page>
);

const Composer = ({
  threadKey,
  sender,
}: {
  threadKey: string;
  sender: Sender;
}): ReactElement => {
  const [text, setText] = useState("");
  const action = useAction(
    useCallback(async () => {
      if (text.trim() === "") return;
      await post(`/api/portal/support/threads/${threadKey}/reply`, { message: text });
      setText("");
      await reload();
    }, [threadKey, text]),
  );
  return (
    <Flex.Box y gap="small" className="portal-composer" full="x">
      <Input.Text
        area
        value={text}
        onChange={setText}
        placeholder={sender === "staff" ? "Reply as Synnax Labs" : "Write a reply"}
        status={action.error != null ? "error" : undefined}
      />
      <Flex.Box x justify="between" align="center">
        <Text.Text level="small" status="error">
          {action.error ?? ""}
        </Text.Text>
        <Button.Button
          variant="filled"
          onClick={action.run}
          disabled={text.trim() === ""}
          status={action.loading ? "loading" : undefined}
          trigger={["Control", "Enter"]}
          triggerIndicator
        >
          Send
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  );
};
