// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/agent/editor/Editor.css";

import { Agent as AgentFlux, Button, Flex, Icon, Input, Text } from "@synnaxlabs/pluto";
import { useCallback, useEffect, useRef, useState } from "react";

import { LoadingMessage, Message } from "@/agent/editor/Message";
import { Arc } from "@/arc";
import { useTask } from "@/arc/hooks";
import { CSS } from "@/css";
import { Controls as Base } from "@/hardware/common/task/controls";
import { type Layout } from "@/layout";
import { usePlacer } from "@/layout/usePlacer";

export const LAYOUT_TYPE = "agent_editor";

export interface CreateArgs {
  key: string;
  name?: string;
}

export const create =
  ({ key, name = "Agent" }: CreateArgs): Layout.Creator =>
  () => ({
    key,
    name,
    icon: "Auto",
    location: "mosaic",
    type: LAYOUT_TYPE,
    window: { navTop: true, showTitle: true },
  });

const SUGGESTIONS = ["Monitor a channel", "Alert on anomalies", "Control a device"];

export const Editor: Layout.Renderer = ({ layoutKey }) => {
  const placeLayout = usePlacer();
  const { data: agentData } = AgentFlux.useRetrieve({ key: layoutKey });
  const send = AgentFlux.useSend();
  const [input, setInput] = useState("");
  const [expanded, setExpanded] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const loading = send.variant === "loading";
  const error = send.variant === "error" ? send.status.message : null;

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [agentData?.messages]);

  const arcKey = agentData?.arcKey ?? "";
  const zeroUUID = "00000000-0000-0000-0000-000000000000";
  const hasArc = arcKey !== "" && arcKey !== zeroUUID;
  const { running, onStartStop, taskStatus } = useTask(
    arcKey,
    agentData?.name ?? "Agent",
  );

  const handleSend = useCallback(async () => {
    if (input.trim().length === 0 || loading) return;
    const content = input.trim();
    setInput("");
    await send.updateAsync({ key: layoutKey, content });
  }, [layoutKey, input, loading, send]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        void handleSend();
      }
    },
    [handleSend],
  );

  const handleViewCode = useCallback(() => {
    if (!hasArc) return;
    placeLayout(
      Arc.Editor.create({
        key: arcKey,
        name: `Code: ${agentData?.name ?? "Agent"}`,
        mode: "text",
        remoteCreated: true,
      }),
    );
  }, [arcKey, hasArc, agentData?.name, placeLayout]);

  const handleToggle = useCallback(() => setExpanded((prev) => !prev), []);
  const handleContract = useCallback(() => setExpanded(false), []);

  const messages = agentData?.messages ?? [];

  return (
    <div className={CSS.B("agent-editor")}>
      <div className={CSS.BE("agent-editor", "messages")}>
        {messages.length === 0 && !loading && (
          <Flex.Box y center grow className={CSS.BE("agent-editor", "empty")}>
            <Icon.Auto className={CSS.BE("agent-editor", "empty-icon")} />
            <Text.Text level="h4" weight={500} color={8}>
              What would you like this agent to do?
            </Text.Text>
            <Text.Text level="p" color={7}>
              Describe a task and the agent will generate Arc code.
            </Text.Text>
            <Flex.Box x gap="small" className={CSS.BE("agent-editor", "chips")}>
              {SUGGESTIONS.map((s) => (
                <Button.Button
                  key={s}
                  variant="outlined"
                  size="small"
                  rounded
                  onClick={() => setInput(s)}
                >
                  {s}
                </Button.Button>
              ))}
            </Flex.Box>
          </Flex.Box>
        )}
        {messages.map((msg, i) => (
          <Message key={i} message={msg} />
        ))}
        {loading && <LoadingMessage />}
        <div ref={messagesEndRef} />
      </div>

      {error != null && (
        <Flex.Box style={{ padding: "0 3rem 1rem" }}>
          <Text.Text level="small" status="error">
            {error}
          </Text.Text>
        </Flex.Box>
      )}

      <Flex.Box y empty className={CSS.BE("agent-editor", "bottom")}>
        <div className={CSS.BE("agent-editor", "input-area")}>
          <div className={CSS.BE("agent-editor", "input-container")}>
            <Input.Text
              area
              full="x"
              value={input}
              onChange={setInput}
              onKeyDown={handleKeyDown}
              placeholder={
                messages.length === 0
                  ? "Describe what you want the agent to do..."
                  : "Refine the agent..."
              }
              className={CSS.BE("agent-editor", "textarea")}
              disabled={loading}
            />
            <Button.Button
              className={CSS.BE("agent-editor", "send-btn")}
              variant="filled"
              size="large"
              square
              rounded
              onClick={() => void handleSend()}
              disabled={input.trim().length === 0 || loading}
            >
              <Icon.Arrow.Up />
            </Button.Button>
          </div>
          {messages.length === 0 && input.length === 0 && (
            <Text.Text
              level="small"
              color={7}
              className={CSS.BE("agent-editor", "hint")}
            >
              Enter to send, Shift+Enter for new line
            </Text.Text>
          )}
        </div>
        {hasArc && (
          <Base.Frame expanded={expanded} onContract={handleContract}>
            <Base.Status
              status={taskStatus}
              expanded={expanded}
              onToggle={handleToggle}
              fallbackMessage="Not deployed"
            />
            <Base.Actions>
              <Button.Button variant="text" size="small" onClick={handleViewCode}>
                View Code
              </Button.Button>
              <Base.StartStopButton running={running} onClick={onStartStop} />
            </Base.Actions>
          </Base.Frame>
        )}
      </Flex.Box>
    </div>
  );
};
