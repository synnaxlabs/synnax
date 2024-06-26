// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, List, Observe, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Layout } from "@/layout";

const Content = (): ReactElement => {
  const client = Synnax.use();

  const [tasks, setTasks] = useState<task.Task[] | undefined>(undefined);

  useQuery({
    queryKey: [client?.key, "tasks"],
    queryFn: async () => {
      if (client == null) return;
      const v = (await client.hardware.tasks.list()).filter((t) => !t.internal);
      setTasks(v);
    },
  });

  Observe.useListener({
    key: [client?.key, "tasks"],
    open: async () => {
      if (client == null) return;
      return client.hardware.tasks.openStateObserver();
    },
    onChange: async (state) => {
      console.log("STATE", state);
      setTasks((prev) => {
        if (prev == null) return;
        const task = prev.find((t) => t.key === state.task);
        if (task != null) task.state = state;
        return [...prev];
      });
    },
  });

  return (
    <Align.Space empty style={{ height: "100%" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Task />}>Tasks</ToolbarTitle>
      </ToolbarHeader>
      <List.List data={tasks}>
        <List.Core<string, task.Task>>
          {(props) => <TaskListTem {...props} />}
        </List.Core>
      </List.List>
    </Align.Space>
  );
};

export const Toolbar: Layout.NavDrawerItem = {
  key: "task",
  icon: <Icon.Task />,
  content: <Content />,
  tooltip: "Tasks",
  initialSize: 300,
  minSize: 225,
  maxSize: 400,
};

export const TaskListTem = (props: List.ItemProps<string, task.Task>) => {
  const { entry } = props;
  const logo = entry.type.includes("ni") ? <Icon.Logo.NI /> : <Icon.Task />;
  return (
    <List.ItemFrame {...props} justify="spaceBetween" align="center">
      <Align.Space direction="y">
        <Align.Space direction="x" align="center">
          <Status.Circle
            variant={(entry.state?.variant as Status.Variant) ?? "info"}
            style={{ fontSize: "2rem" }}
          />
          <Text.WithIcon level="p" startIcon={logo} weight={500}>
            {entry.name}
          </Text.WithIcon>
        </Align.Space>
        <Text.Text level="small" shade={6}>
          {entry.type}
        </Text.Text>
      </Align.Space>
      <Button.Icon
        variant="outlined"
        onClick={() => {
          entry.executeCommand(
            entry.state?.details?.running === true ? "stop" : "start",
          );
        }}
      >
        {entry.state?.details?.running === true ? <Icon.Pause /> : <Icon.Play />}
      </Button.Icon>
    </List.ItemFrame>
  );
};
