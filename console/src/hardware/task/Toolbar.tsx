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
  Menu as PMenu,
  Observe,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { errors, strings } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Menu } from "@/components/menu";
import { Confirm } from "@/confirm";
import { createTaskLayout } from "@/hardware/task/ontology";
import { Layout } from "@/layout";

const Content = (): ReactElement => {
  const client = Synnax.use();
  const [tasks, setTasks] = useState<task.Task[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const menuProps = PMenu.useContextMenu();
  const addStatus = Status.useAggregator();
  const dispatch = useDispatch();
  const placer = Layout.usePlacer();

  useAsyncEffect(async () => {
    if (client == null) return;
    const v = (await client.hardware.tasks.list({ includeState: true })).filter(
      (t) => !t.internal,
    );
    setTasks(v);
  }, [client]);

  Observe.useListener({
    key: [client?.key, "tasks.state"],
    open: async () => client?.hardware.tasks.openStateObserver(),
    onChange: (state) =>
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
      client.hardware.tasks.retrieve(addedOrUpdated).then((nextTasks) => {
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

  const confirm = Confirm.useModal();

  const handleDelete = useMutation<void, Error, string[], task.Task[]>({
    mutationFn: async (keys: string[]) => {
      setSelected([]);
      if (client == null) throw new Error("Client not available");
      const deletedNames = tasks
        .filter((task) => keys.includes(task.key))
        .map((task) => task.name);
      const names = strings.naturalLanguageJoin(deletedNames, "tasks");
      if (
        !(await confirm({
          message: `Are you sure you want to delete ${names}?`,
          description: "This action cannot be undone.",
          cancel: { label: "Cancel" },
          confirm: { label: "Delete", variant: "error" },
        }))
      )
        return;
      await client.hardware.tasks.delete(keys.map((k) => BigInt(k)));
      dispatch(Layout.remove({ keys }));
      setTasks((prev) => {
        const next = prev.filter((t) => !keys.includes(t.key.toString()));
        return [...next];
      });
    },
    onError: (e) => {
      if (errors.CANCELED.matches(e)) return;
      addStatus({
        variant: "error",
        message: "Failed to delete tasks",
        description: e.message,
      });
    },
  });

  return (
    <PMenu.ContextMenu
      menu={({ keys }) => {
        const selected = keys.map((k) => tasks.find((t) => t.key === k));
        const canStart = selected.some((t) => t?.state?.details?.running !== true);
        const canStop = selected.some((t) => t?.state?.details?.running === true);
        const someSelected = selected.length > 0;
        const isSingle = selected.length === 1;
        const handleEdit = (key: string): void => {
          const type = tasks.find((t) => t.key === key)?.type;
          if (type == null)
            return addStatus({
              variant: "error",
              message: "Failed to open task details",
              description: `Task with key ${key} not found`,
            });
          const layout = createTaskLayout(key, type);
          placer(layout);
        };
        return (
          <PMenu.Menu
            level="small"
            iconSpacing="small"
            onChange={{
              delete: () => handleDelete.mutate(keys),
              start: () => selected.forEach((t) => t?.executeCommand("start")),
              stop: () => selected.forEach((t) => t?.executeCommand("stop")),
              edit: () => handleEdit(keys[0]),
            }}
          >
            {canStart && (
              <PMenu.Item startIcon={<Icon.Play />} itemKey="start">
                {isSingle ? "Start Task" : "Start Tasks"}
              </PMenu.Item>
            )}
            {canStop && (
              <PMenu.Item startIcon={<Icon.Pause />} itemKey="stop">
                {isSingle ? "Stop Task" : "Stop Tasks"}
              </PMenu.Item>
            )}
            {canStart || (canStop && <PMenu.Divider />)}
            {isSingle && (
              <PMenu.Item startIcon={<Icon.Edit />} itemKey="edit">
                Edit Configuration
              </PMenu.Item>
            )}
            {someSelected && (
              <>
                <PMenu.Item startIcon={<Icon.Delete />} itemKey="delete">
                  Delete
                </PMenu.Item>
                <PMenu.Divider />
              </>
            )}
            <Menu.HardReloadItem />
          </PMenu.Menu>
        );
      }}
      {...menuProps}
    >
      <Align.Space empty style={{ height: "100%" }}>
        <ToolbarHeader>
          <ToolbarTitle icon={<Icon.Task />}>Tasks</ToolbarTitle>
        </ToolbarHeader>
        <List.List data={tasks}>
          <List.Selector value={selected} onChange={setSelected} replaceOnSingle>
            <List.Core<string, task.Task>>
              {({ key, ...props }) => <TaskListItem key={key} {...props} />}
            </List.Core>
          </List.Selector>
        </List.List>
      </Align.Space>
    </PMenu.ContextMenu>
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

const TaskListItem = (props: List.ItemProps<string, task.Task>) => {
  const {
    entry,
    entry: { type },
  } = props;
  const logo = type.startsWith("ni") ? (
    <Icon.Logo.NI />
  ) : type.startsWith("opc") ? (
    <Icon.Logo.OPC />
  ) : (
    <Icon.Task />
  );

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
        onClick={() =>
          entry.executeCommand(
            entry.state?.details?.running === true ? "stop" : "start",
          )
        }
      >
        {entry.state?.details?.running === true ? <Icon.Pause /> : <Icon.Play />}
      </Button.Icon>
    </List.ItemFrame>
  );
};
