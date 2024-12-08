// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align, Button, Header, Input, Status, Text } from "@synnaxlabs/pluto";
import { caseconv, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";
import { useDispatch } from "react-redux";

import { type LogMessage, useLogsContext } from "@/cluster/embedded/LogsProvider";
import { ICON_MAP, LEVEL_COLORS, STATUS_MAP } from "@/cluster/embedded/types";
import { useSelectEmbeddedState } from "@/cluster/selectors";
import { setLocalState } from "@/cluster/slice";
import { type Layout } from "@/layout";

const logElement = (v: LogMessage): ReactElement => {
  const { level, msg, ts, error } = v;
  return (
    <Align.Space direction="x" size="small" key={`${ts}-${msg}`}>
      <Text.Text level="p" noWrap shade={7} style={{ width: 90, flexShrink: 0 }}>
        {TimeStamp.seconds(ts).fString("preciseTime")}{" "}
      </Text.Text>
      <Text.Text
        level="p"
        color={LEVEL_COLORS[level]}
        style={{ width: 50, flexShrink: 0 }}
      >
        {level.toUpperCase()}
      </Text.Text>{" "}
      <Text.Text level="p" style={{ flexGrow: 0 }}>
        {caseconv.capitalize(msg)} {error}
      </Text.Text>
    </Align.Space>
  );
};

export const Controls: Layout.Renderer = () => {
  const [logs, setLogs] = useState<ReactElement[]>([]);
  const d = useDispatch();
  const { status, pid } = useSelectEmbeddedState();
  const { subscribeToLogs, getLogs } = useLogsContext();
  const addStatus = Status.useAggregator();
  useEffect(() => {
    const [logs, d] = subscribeToLogs((v) => {
      const log = logElement(v);
      if (log != null)
        setLogs((logs) => {
          const newLogs = [log, ...logs];
          return newLogs.slice(0, 100);
        });
    });
    setLogs(
      logs
        .map(logElement)
        .filter((v) => v != null)
        .reverse(),
    );
    return d;
  }, []);
  const handleStopStart = () => {
    d(setLocalState({ command: status == "running" ? "stop" : "start" }));
  };

  const handleKill = () => {
    d(setLocalState({ command: "kill" }));
  };

  return (
    <Align.Space direction="y" style={{ padding: "3rem" }} size="large" grow>
      <Text.Text level="h3" weight={450}>
        Embedded
      </Text.Text>
      <Align.Space direction="x" justify="spaceBetween" style={{ padding: "0 1rem" }}>
        <Align.Space direction="x" size="large">
          <Input.Item label="Status">
            <Status.Text variant={STATUS_MAP[status]} level="p" style={{ width: 80 }}>
              {caseconv.capitalize(status)}
            </Status.Text>
          </Input.Item>
          <Input.Item label="PID">
            <Text.Text level="p">{pid}</Text.Text>
          </Input.Item>
        </Align.Space>
        <Align.Space direction="x">
          <Button.Button status="error" startIcon={<Icon.Stop />} onClick={handleKill}>
            Kill
          </Button.Button>
          <Button.Button
            variant="outlined"
            disabled={status == "starting" || status == "stopping"}
            startIcon={ICON_MAP[status]}
            onClick={handleStopStart}
          >
            {status == "running" ? "Stop" : "Start"}
          </Button.Button>
        </Align.Space>
      </Align.Space>
      <Align.Space
        direction="y"
        rounded
        bordered
        background={1}
        empty
        style={{ overflow: "hidden" }}
        grow
      >
        <Header.Header level="h5">
          <Header.Title>Logs</Header.Title>
          <Header.Actions>
            {[
              {
                key: "copy",
                children: <Icon.Copy />,
                onClick: () => {
                  navigator.clipboard.writeText(
                    getLogs()
                      .map((v) => JSON.stringify(v))
                      .join("\n"),
                  );
                  addStatus({
                    variant: "info",
                    message: "Logs copied to clipboard",
                  });
                },
              },
            ]}
          </Header.Actions>
        </Header.Header>
        <Align.Space
          direction="y"
          style={{
            overflowY: "auto",
            overflowX: "hidden",
            padding: "2rem",
            height: "100%",
          }}
          empty
          grow
        >
          {logs}
        </Align.Space>
      </Align.Space>
    </Align.Space>
  );
};
