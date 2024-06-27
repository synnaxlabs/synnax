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
import {
  Align,
  Button,
  List,
  Menu,
  Observe,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Layout } from "@/layout";

const Content = (): ReactElement => {
  const client = Synnax.use();

  const [tasks, setTasks] = useState<task.Task[]>([]);

  useAsyncEffect(async () => {
    if (client == null) return;
    const v = (await client.hardware.tasks.list()).filter((t) => !t.internal);
    setTasks(v);
  }, [client]);

  Observe.useListener({
    key: [client?.key, "tasks.state"],
    open: async () => client?.hardware.tasks.openStateObserver(),
    onChange: async (state) =>
      setTasks((prev) => {
        const task = prev.find((t) => t.key === state.task);
        if (task != null) task.state = state;
        return [...prev];
      }),
  });

  Observe.useListener({
    key: [client?.key, "tasks.updates"],
    open: async () => client?.hardware.tasks.openTracker(),
    onChange: (update) => {
      if (client == null) return;
      const removed = update.filter((u) => u.variant === "delete").map((u) => u.key);
      const addedOrUpdated = update
        .filter((u) => u.variant === "set")
        .map((u) => u.key);
      client?.hardware.tasks.retrieve(addedOrUpdated).then((nextTasks) => {
        setTasks((prev) => {
          const next = prev
            .filter((t) => !removed.includes(t.key))
            .map((t) => {
              const u = nextTasks.find((u) => u.key === t.key);
              if (u == null) return t;
              u.state = t.state;
              return u;
            });
          const nextKeys = next.map((t) => t.key);
          return [
            ...next,
            ...nextTasks.filter((u) => !u.internal && !nextKeys.includes(u.key)),
          ];
        });
      });
    },
  });

  const [selected, setSelected] = useState<string[]>([]);

  const menuProps = Menu.useContextMenu();

  const addStatus = Status.useAggregator();
  const dispatch = useDispatch();

  const del = useMutation<void, Error, string[], task.Task[]>({
    onMutate: (keys: string[]) => {
      setSelected([]);
      const toDelete: task.Task[] = [];
      setTasks((prev) => {
        const next = prev.filter((t) => {
          const includes = keys.includes(t.key.toString());
          if (includes) toDelete.push(t);
          return !includes;
        });
        return [...next];
      });
      return toDelete;
    },
    mutationFn: async (keys: string[]) => {
      if (client == null) return;
      await client.hardware.tasks.delete(keys.map((k) => BigInt(k)));
      dispatch(Layout.remove({ keys }));
      setSelected([]);
    },
    onError: ({ message }, _, toDelete) => {
      addStatus({
        variant: "error",
        message: "Failed to delete tasks",
        description: message,
      });
      if (toDelete != null) setTasks((prev) => [...prev, ...toDelete]);
    },
  });

  return (
    <Menu.ContextMenu
      menu={({ keys }) => {
        const selected = keys.map((k) => tasks.find((t) => t.key === k));
        const canStart = selected.some((t) => t?.state?.details?.running !== true);
        const canStop = selected.some((t) => t?.state?.details?.running === false);
        return (
          <Menu.Menu
            level="small"
            iconSpacing="small"
            onChange={{
              delete: () => del.mutate(keys),
            }}
          >
            {canStart && (
              <Menu.Item startIcon={<Icon.Play />} itemKey="start">
                Start Tasks
              </Menu.Item>
            )}
            {canStop && (
              <Menu.Item startIcon={<Icon.Pause />} itemKey="stop">
                Stop Tasks
              </Menu.Item>
            )}
            <Menu.Divider />
            <Menu.Item startIcon={<Icon.Delete />} itemKey="delete">
              Delete
            </Menu.Item>
          </Menu.Menu>
        );
      }}
      {...menuProps}
    >
      <Align.Space empty style={{ height: "100%" }}>
        <ToolbarHeader>
          <ToolbarTitle icon={<Icon.Task />}>Tasks</ToolbarTitle>
        </ToolbarHeader>
        <List.List data={tasks}>
          <List.Selector value={selected} onChange={setSelected}>
            <List.Core<string, task.Task>>
              {(props) => <TaskListTem {...props} />}
            </List.Core>
          </List.Selector>
        </List.List>
      </Align.Space>
    </Menu.ContextMenu>
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
    <List.ItemFrame {...props} justify="spaceBetween" align="center" rightAligned>
      <Align.Space direction="y" size="small">
        <Align.Space direction="x" align="center">
          <Status.Circle
            variant={(entry.state?.variant as Status.Variant) ?? "info"}
            style={{ fontSize: "2rem" }}
          />
          <Text.WithIcon level="p" startIcon={logo} weight={500} noWrap>
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
