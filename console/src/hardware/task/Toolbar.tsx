// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/task/Toolbar.css";

import { type task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Header,
  List,
  Menu as PMenu,
  Observe,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
  useDelayedState,
} from "@synnaxlabs/pluto";
import { errors, strings } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Menu } from "@/components/menu";
import { Confirm } from "@/confirm";
import { CSS } from "@/css";
import { checkDesiredStateMatch } from "@/hardware/task/common/useDesiredState";
import { createLayout } from "@/hardware/task/ontology";
import { createSelector } from "@/hardware/task/Selector";
import { getIcon, parseType } from "@/hardware/task/types";
import { Layout } from "@/layout";

type DesiredTaskState = "running" | "paused" | null;

const EmptyContent = (): ReactElement => {
  const place = Layout.usePlacer();
  const handleClick: React.MouseEventHandler<HTMLParagraphElement> = (e) => {
    e.stopPropagation();
    place(createSelector({}));
  };
  return (
    <Align.Space empty style={{ height: "100%", position: "relative" }}>
      <Align.Center direction="y" style={{ height: "100%" }} size="small">
        <Text.Text level="p">No existing tasks.</Text.Text>
        <Text.Link level="p" onClick={handleClick}>
          Add a task
        </Text.Link>
      </Align.Center>
    </Align.Space>
  );
};

interface MutationVars {
  name: string;
  key: string;
}

const Content = (): ReactElement => {
  const client = Synnax.use();
  const [tasks, setTasks] = useState<task.Task[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const handleRename = useMutation<void, Error, MutationVars, string>({
    mutationKey: ["renameTask"],
    onMutate: ({ name, key }) => {
      const oldTask = tasks.find((t) => t.key === key);
      const oldName = oldTask?.name;
      if (oldTask == null) return oldName;
      setTasks((prev) => {
        const t = prev.find((t) => t.key === key);
        if (t != null) t.name = name;
        return [...prev];
      });
      return oldName;
    },
    mutationFn: async ({ name, key }) => {
      const tsk = tasks.find((t) => t.key === key);
      const isRunning = tsk?.state?.details?.running === true;
      if (
        isRunning &&
        !(await confirm({
          message: `Are you sure you want to rename the task to ${name}?`,
          description: "This will cause the task to stop and be reconfigured.",
          cancel: { label: "Cancel" },
          confirm: { label: "Rename", variant: "error" },
        }))
      )
        return;
      if (client == null) throw new Error("Client not available");
      const t = await client.hardware.tasks.retrieve(key);
      if (t == null) return;
      await client.hardware.tasks.create({ ...t, name });
    },
    onError: (e, { name, key }, oldName) => {
      if (oldName != null)
        setTasks((prev) => {
          const t = prev.find((t) => t.key === key);
          if (t != null) t.name = oldName;
          return [...prev];
        });
      if (errors.CANCELED.matches(e)) return;
      addStatus({
        variant: "error",
        message: `Failed to rename ${oldName ?? "task"} to ${name}`,
        description: e.message,
      });
    },
  }).mutate;
  const [desiredStates, setDesiredStates] = useState<
    Record<task.TaskKey, DesiredTaskState>
  >({});
  const menuProps = PMenu.useContextMenu();
  const addStatus = Status.useAggregator();
  const dispatch = useDispatch();
  const place = Layout.usePlacer();
  useAsyncEffect(async () => {
    if (client == null) return;
    const v = (await client.hardware.tasks.list({ includeState: true })).filter(
      (t) => !t.internal && !t.snapshot,
    );
    setTasks(v);
  }, [client]);
  Observe.useListener({
    key: [client?.key],
    open: async () => client?.hardware.tasks.openStateObserver(),
    onChange: (state) => {
      const key = state.task;
      setTasks((prev) => {
        const task = prev.find((t) => t.key === key);
        if (task != null) task.state = state;
        return [...prev];
      });
      const nowRunning = state.details?.running;
      const newState =
        nowRunning === true ? "running" : nowRunning === false ? "paused" : null;
      if (newState === desiredStates[key]) return;
      setDesiredStates((prev) => {
        const next = { ...prev };
        next[key] = newState;
        return next;
      });
    },
  });
  Observe.useListener({
    key: [client?.key],
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
            ...nextTasks.filter(
              (u) => !u.internal && !u.snapshot && !nextKeys.includes(u.key),
            ),
          ];
        });
      });
    },
  });
  Observe.useListener({
    key: [client?.key],
    open: async () => client?.hardware.tasks.openCommandObserver(),
    onChange: (command) => {
      const type = command.type;
      if (type !== "start" && type !== "stop") return;
      const nextState = type === "start" ? "running" : "paused";
      const task = command.task;
      if (desiredStates[task] === nextState) return;
      setDesiredStates((prev) => {
        const next = { ...prev };
        next[task] = nextState;
        return next;
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
      setDesiredStates((prev) => {
        const next = { ...prev };
        keys.forEach((k) => delete next[k]);
        return next;
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
          const task = tasks.find((t) => t.key === key);
          if (task == null)
            return addStatus({
              variant: "error",
              message: "Failed to open task details",
              description: `Task with key ${key} not found`,
            });
          const layout = createLayout(task);
          place(layout);
        };
        return (
          <PMenu.Menu
            level="small"
            iconSpacing="small"
            onChange={{
              rename: () => Text.edit(`text-${keys[0]}`),
              delete: () => handleDelete.mutate(keys),
              start: () =>
                selected.forEach((t) => {
                  if (t == null) return;
                  t.executeCommand("start");
                  if (desiredStates[t.key] === "running") return;
                  setDesiredStates((prev) => {
                    t.executeCommand("start");
                    const next = { ...prev };
                    next[t.key] = "running";
                    return next;
                  });
                }),
              stop: () =>
                selected.forEach((t) => {
                  if (t == null) return;
                  t.executeCommand("stop");
                  if (desiredStates[t.key] === "paused") return;
                  setDesiredStates((prev) => {
                    const next = { ...prev };
                    next[t.key] = "paused";
                    return next;
                  });
                }),
              edit: () => handleEdit(keys[0]),
            }}
          >
            {canStart && (
              <PMenu.Item startIcon={<Icon.Play />} itemKey="start">
                Start
              </PMenu.Item>
            )}
            {canStop && (
              <PMenu.Item startIcon={<Icon.Pause />} itemKey="stop">
                Stop
              </PMenu.Item>
            )}
            {(canStart || canStop) && <PMenu.Divider />}
            {isSingle && (
              <>
                <PMenu.Item startIcon={<Icon.Edit />} itemKey="edit">
                  Edit Configuration
                </PMenu.Item>
                <PMenu.Divider />
                <Menu.RenameItem />
              </>
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
      <Align.Space empty style={{ height: "100%" }} className={CSS.B("task-toolbar")}>
        <ToolbarHeader>
          <ToolbarTitle icon={<Icon.Task />}>Tasks</ToolbarTitle>
          <Header.Actions>
            {[
              {
                children: <Icon.Add />,
                onClick: () => place(createSelector({})),
              },
            ]}
          </Header.Actions>
        </ToolbarHeader>
        <List.List data={tasks} emptyContent={<EmptyContent />}>
          <List.Selector value={selected} onChange={setSelected} replaceOnSingle>
            <List.Core<string, task.Task>>
              {({ key, ...props }) => (
                <TaskListItem
                  key={key}
                  {...props}
                  desiredState={desiredStates[props.entry.key]}
                  onStopStart={(state) => {
                    if (state == null) return;
                    if (desiredStates[props.entry.key] === state) return;
                    setDesiredStates((prev) => {
                      const next = { ...prev };
                      next[props.entry.key] = state;
                      return next;
                    });
                  }}
                  onRename={(name) => handleRename({ name, key })}
                />
              )}
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

interface TaskListItemProps extends List.ItemProps<string, task.Task> {
  desiredState: DesiredTaskState;
  onStopStart: (state: DesiredTaskState) => void;
  onRename: (name: string) => void;
}

const TaskListItem = ({
  desiredState,
  onStopStart,
  onRename,
  ...props
}: TaskListItemProps) => {
  const {
    entry,
    entry: { type, state },
  } = props;
  const logo = getIcon(type);
  const isRunning = entry.state?.details?.running === true;
  const isLoading =
    desiredState != null &&
    !checkDesiredStateMatch(desiredState, isRunning) &&
    state?.variant === "success";
  const loading = useDelayedState<boolean>(false, isLoading);
  const handleClick = () => {
    onStopStart(isRunning ? "paused" : "running");
    entry.executeCommand(isRunning ? "stop" : "start");
  };
  return (
    <List.ItemFrame {...props} justify="spaceBetween" align="center" rightAligned>
      <Align.Space
        direction="y"
        size="small"
        grow
        className={CSS.BE("task", "meta-data")}
      >
        <Align.Space direction="x" align="center" size="small">
          <Status.Circle
            variant={Status.VARIANTS.find((v) => v === state?.variant)}
            style={{ fontSize: "2rem", minWidth: "2rem" }}
          />
          <Text.WithIcon
            className={CSS.BE("task", "title")}
            level="p"
            startIcon={logo}
            weight={500}
            noWrap
          >
            <Text.MaybeEditable
              id={`text-${entry.key}`}
              level="p"
              value={entry.name}
              onChange={onRename}
              allowDoubleClick={false}
            />
          </Text.WithIcon>
        </Align.Space>
        <Text.Text level="small" shade={6}>
          {parseType(type)}
        </Text.Text>
      </Align.Space>
      <Button.Icon
        variant="outlined"
        loading={loading}
        onClick={handleClick}
        tooltip={`${isRunning ? "Stop" : "Start"} ${entry.name}`}
      >
        {isRunning ? <Icon.Pause /> : <Icon.Play />}
      </Button.Icon>
    </List.ItemFrame>
  );
};
