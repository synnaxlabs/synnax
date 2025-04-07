// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/task/Toolbar.css";

import { task, UnexpectedError } from "@synnaxlabs/client";
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
import { errors, strings, TimeSpan, type UnknownRecord } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Menu, Toolbar } from "@/components";
import { CSS } from "@/css";
import { NULL_CLIENT_ERROR } from "@/errors";
import { Common } from "@/hardware/common";
import { createLayout } from "@/hardware/task/layouts";
import { SELECTOR_LAYOUT } from "@/hardware/task/Selector";
import { getIcon, parseType } from "@/hardware/task/types";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Modals } from "@/modals";

interface SugaredDetails extends UnknownRecord {
  status: Common.Task.Status;
}

interface SugaredState extends task.State {
  details: SugaredDetails;
}

interface SugaredTask extends task.Task {
  state: SugaredState;
}

const sugarTask = (task: task.Task): SugaredTask => {
  if (task.state?.details?.status != null) return task as SugaredTask;

  if (task.state?.details != null) {
    task.state.details = {
      ...task.state.details,
      status: task.state.details.running
        ? Common.Task.RUNNING_STATUS
        : Common.Task.PAUSED_STATUS,
    };
    return task as SugaredTask;
  }

  if (task.state != null) {
    task.state = { ...task.state, details: { status: Common.Task.PAUSED_STATUS } };
    return task as SugaredTask;
  }

  const state: SugaredState = {
    variant: "success",
    task: task.key,
    details: { status: Common.Task.PAUSED_STATUS },
  };
  return Object.assign(task, { state });
};

const updateTaskStatus = (tsk: SugaredTask, state: task.State): SugaredTask => {
  const running = state.details?.running;
  const newStatus =
    running === true
      ? Common.Task.RUNNING_STATUS
      : running === false
        ? Common.Task.PAUSED_STATUS
        : tsk.state.details.status;
  tsk.state = {
    ...tsk.state,
    ...state,
    details: { ...tsk.state.details, ...state.details, status: newStatus },
  };
  return tsk;
};

const setLoading = (task: SugaredTask): SugaredTask => {
  task.state.details.status = Common.Task.LOADING_STATUS;
  return task;
};

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  const handleClick = () => {
    placeLayout(SELECTOR_LAYOUT);
  };
  return (
    <Align.Space empty style={{ height: "100%", position: "relative" }}>
      <Align.Center y style={{ height: "100%" }} size="small">
        <Text.Text level="p">No existing tasks.</Text.Text>
        <Text.Link level="p" onClick={handleClick}>
          Add a task
        </Text.Link>
      </Align.Center>
    </Align.Space>
  );
};

interface RenameArgs {
  name: string;
  key: task.Key;
}

interface StartStopArgs {
  command: Common.Task.StartOrStopCommand;
  keys: task.Key[];
}

const Content = () => {
  const client = Synnax.use();
  const [tasks, setTasks] = useState<SugaredTask[]>([]);
  const [selected, setSelected] = useState<task.Key[]>([]);
  const handleError = Status.useErrorHandler();
  const rename = useMutation({
    onMutate: ({ key }) => tasks.find((t) => t.key === key)?.name ?? "task",
    mutationFn: async ({ name, key }: RenameArgs) => {
      const tsk = tasks.find((t) => t.key === key);
      if (tsk == null) throw new UnexpectedError(`Task with key ${key} not found`);
      if (tsk.state.details.status === Common.Task.RUNNING_STATUS) {
        const confirmed = await confirm({
          message: `Are you sure you want to rename ${tsk.name} to ${name}?`,
          description: `This will cause ${tsk.name} to stop and be reconfigured.`,
          cancel: { label: "Cancel" },
          confirm: { label: "Rename", variant: "error" },
        });
        if (!confirmed) return;
      }
      dispatch(Layout.rename({ key, name }));
      setTasks((prev) =>
        prev.map((task) => {
          if (task.key === key) task.name = name;
          return task;
        }),
      );
      if (client == null) throw NULL_CLIENT_ERROR;
      await client.hardware.tasks.create({ ...tsk, name });
    },
    onError: (e, { name, key }, oldName) => {
      if (oldName != null)
        setTasks((prev) =>
          prev.map((tsk) => {
            if (tsk.key === key) tsk.name = oldName;
            return tsk;
          }),
        );
      handleError(e, `Failed to rename ${oldName ?? "task"} to ${name}`);
    },
  }).mutate;
  const menuProps = PMenu.useContextMenu();
  const dispatch = useDispatch();
  const placeLayout = Layout.usePlacer();
  useAsyncEffect(async () => {
    if (client == null) {
      setTasks([]);
      return;
    }
    const allTasks = await client.hardware.tasks.list({ includeState: true });
    const shownTasks = allTasks.filter(
      ({ internal, snapshot }) => !internal && !snapshot,
    );
    setTasks(shownTasks.map(sugarTask));
  }, [client?.key]);
  Observe.useListener({
    key: [client?.key],
    open: async () => client?.hardware.tasks.openStateObserver(),
    onChange: (state) => {
      const key = state.task;
      setTasks((prev) => {
        const tsk = prev.find((t) => t.key === key);
        if (tsk == null) return prev;
        updateTaskStatus(tsk, state);
        return [...prev];
      });
    },
  });
  Observe.useListener({
    key: [client?.key],
    open: async () => client?.hardware.tasks.openTracker(),
    onChange: (update) => {
      if (client == null) return;
      const removed = new Set(
        update.filter(({ variant }) => variant === "delete").map(({ key }) => key),
      );
      const addedOrUpdated = update
        .filter(({ variant }) => variant === "set")
        .map(({ key }) => key);
      handleError(async () => {
        const changedTasks = await client.hardware.tasks.retrieve(addedOrUpdated, {
          includeState: true,
        });
        const sugaredChangedTasks = changedTasks
          .filter(({ internal, snapshot }) => !internal && !snapshot)
          .map(sugarTask);
        const changedTasksMap = new Map<task.Key, SugaredTask>();
        sugaredChangedTasks.forEach((task) => {
          changedTasksMap.set(task.key, task);
        });
        setTasks((prev) => {
          const next = prev
            .filter(({ key }) => !removed.has(key))
            .map((t) => changedTasksMap.get(t.key) ?? t);
          const existingKeys = new Set(next.map(({ key }) => key));
          return [
            ...next,
            ...sugaredChangedTasks.filter(({ key }) => !existingKeys.has(key)),
          ];
        });
        setSelected((prev) => prev.filter((k) => !removed.has(k)));
      }, "Failed to update task toolbar");
    },
  });
  Observe.useListener({
    key: [client?.key],
    open: async () => client?.hardware.tasks.openCommandObserver(),
    onChange: ({ type, task }) => {
      const status = tasks.find(({ key }) => key === task)?.state.details.status;
      if (status == null) return;
      if (Common.Task.shouldExecuteCommand(status, type))
        setTasks((prev) =>
          prev.map((tsk) => {
            if (tsk.key === task) setLoading(tsk);
            return tsk;
          }),
        );
    },
  });
  const confirm = Modals.useConfirm();
  const handleDelete = useMutation({
    mutationFn: async (keys: string[]) => {
      setSelected([]);
      if (keys.length === 0) return;
      if (client == null) throw NULL_CLIENT_ERROR;
      const deletedNames = tasks
        .filter(({ key }) => keys.includes(key))
        .map(({ name }) => name);
      const names = strings.naturalLanguageJoin(deletedNames, "tasks");
      const confirmed = await confirm({
        message: `Are you sure you want to delete ${names}?`,
        description: "This action cannot be undone.",
        cancel: { label: "Cancel" },
        confirm: { label: "Delete", variant: "error" },
      });
      if (!confirmed) return;
      await client.hardware.tasks.delete(keys.map(BigInt));
      dispatch(Layout.remove({ keys }));
      setTasks((prev) => prev.filter(({ key }) => !keys.includes(key)));
    },
    onError: (e) => {
      if (errors.CANCELED.matches(e)) return;
      handleError(e, "Failed to delete tasks");
    },
  }).mutate;
  const actions = useMemo(
    () => [{ children: <Icon.Add />, onClick: () => placeLayout(SELECTOR_LAYOUT) }],
    [placeLayout],
  );
  const startOrStop = useMutation({
    mutationFn: async ({ command, keys }: StartStopArgs) => {
      if (client == null) throw NULL_CLIENT_ERROR;
      const filteredKeys = new Set(
        keys.filter((k) => {
          const status = tasks.find(({ key }) => key === k)?.state.details.status;
          if (status == null) throw new UnexpectedError(`Task with key ${k} not found`);
          return Common.Task.shouldExecuteCommand(status, command);
        }),
      );
      setTasks((prev) =>
        prev.map((tsk) => (filteredKeys.has(tsk.key) ? setLoading(tsk) : tsk)),
      );
      const tasksToExecute = tasks.filter(({ key }) => filteredKeys.has(key));
      tasksToExecute.forEach((t) => {
        t.executeCommandSync(command, {}, TimeSpan.fromSeconds(10)).catch((e) => {
          const status: task.State = {
            variant: "error",
            task: t.key,
            details: { message: e.message },
          };
          setTasks((prev) =>
            prev.map((tsk) =>
              tsk.key === t.key ? updateTaskStatus(tsk, status) : tsk,
            ),
          );
        });
      });
    },
    onError: (e, { command }) => handleError(e, `Failed to ${command} tasks`),
  }).mutate;
  const handleStart = useCallback(
    (keys: string[]) => startOrStop({ command: Common.Task.START_COMMAND, keys }),
    [startOrStop],
  );
  const handleStop = useCallback(
    (keys: string[]) => startOrStop({ command: Common.Task.STOP_COMMAND, keys }),
    [startOrStop],
  );
  const contextMenu = useCallback<NonNullable<PMenu.ContextMenuProps["menu"]>>(
    ({ keys }) => (
      <ContextMenu
        keys={keys}
        tasks={tasks}
        onDelete={handleDelete}
        onStart={handleStart}
        onStop={handleStop}
      />
    ),
    [handleDelete, handleStart, handleStop, tasks],
  );
  const handleListItemStopStart = useCallback(
    (command: Common.Task.StartOrStopCommand, key: task.Key) =>
      startOrStop({ command, keys: [key] }),
    [startOrStop],
  );
  const listItem = useCallback<List.ItemRenderProp<string, SugaredTask>>(
    ({ key, ...p }) => (
      <TaskListItem
        key={key}
        {...p}
        onStopStart={(command) => handleListItemStopStart(command, key)}
        onRename={(name) => rename({ name, key })}
      />
    ),
    [handleListItemStopStart, rename],
  );
  return (
    <>
      <PMenu.ContextMenu menu={contextMenu} {...menuProps} />
      <Align.Space
        empty
        style={{ height: "100%" }}
        className={CSS(CSS.B("task-toolbar"), menuProps.className)}
        onContextMenu={menuProps.open}
      >
        <Toolbar.Header>
          <Toolbar.Title icon={<Icon.Task />}>Tasks</Toolbar.Title>
          <Toolbar.Actions>{actions}</Toolbar.Actions>
        </Toolbar.Header>
        <List.List data={tasks} emptyContent={<EmptyContent />}>
          <List.Selector value={selected} onChange={setSelected} replaceOnSingle>
            <List.Core<task.Key, SugaredTask>>{listItem}</List.Core>
          </List.Selector>
        </List.List>
      </Align.Space>
    </>
  );
};

export const TOOLBAR_NAV_DRAWER_ITEM: Layout.NavDrawerItem = {
  key: "task",
  icon: <Icon.Task />,
  content: <Content />,
  trigger: ["T"],
  tooltip: "Tasks",
  initialSize: 300,
  minSize: 225,
  maxSize: 400,
};

interface TaskListItemProps extends List.ItemProps<task.Key, SugaredTask> {
  onStopStart: (command: Common.Task.StartOrStopCommand) => void;
  onRename: (name: string) => void;
}

const TaskListItem = ({ onStopStart, onRename, ...rest }: TaskListItemProps) => {
  const {
    key,
    name,
    state: {
      details: { status },
      variant,
    },
    type,
  } = rest.entry;
  const icon = getIcon(type);
  const isLoading = status === Common.Task.LOADING_STATUS;
  const isRunning = status === Common.Task.RUNNING_STATUS;
  const handleClick = useCallback<NonNullable<Button.IconProps["onClick"]>>(
    (e) => {
      e.stopPropagation();
      const command = isRunning ? Common.Task.STOP_COMMAND : Common.Task.START_COMMAND;
      onStopStart(command);
    },
    [isRunning, onStopStart],
  );
  return (
    <List.ItemFrame {...rest} justify="spaceBetween" align="center" rightAligned>
      <Align.Space y size="small" grow className={CSS.BE("task", "metadata")}>
        <Align.Space x align="center" size="small">
          <Status.Circle
            variant={status === Common.Task.LOADING_STATUS ? "loading" : variant}
            style={{ fontSize: "2rem", minWidth: "2rem" }}
          />
          <Text.WithIcon
            className={CSS.BE("task", "title")}
            level="p"
            startIcon={icon}
            weight={500}
            noWrap
          >
            <Text.MaybeEditable
              id={`text-${key}`}
              level="p"
              value={name}
              onChange={onRename}
              allowDoubleClick={false}
            />
          </Text.WithIcon>
        </Align.Space>
        <Text.Text level="small" shade={10}>
          {parseType(type)}
        </Text.Text>
      </Align.Space>
      <Button.Icon
        variant="outlined"
        loading={isLoading}
        onClick={handleClick}
        tooltip={`${isRunning ? "Stop" : "Start"} ${name}`}
      >
        {isRunning ? <Icon.Pause /> : <Icon.Play />}
      </Button.Icon>
    </List.ItemFrame>
  );
};

interface ContextMenuProps {
  keys: task.Key[];
  onDelete: (keys: task.Key[]) => void;
  onStart: (keys: task.Key[]) => void;
  onStop: (keys: task.Key[]) => void;
  tasks: SugaredTask[];
}

const ContextMenu = ({ keys, tasks, onDelete, onStart, onStop }: ContextMenuProps) => {
  const selectedKeys = new Set(keys);
  const selectedTasks = tasks.filter(({ key }) => selectedKeys.has(key));

  const canStart = selectedTasks.some(
    ({
      state: {
        details: { status },
      },
    }) => status === Common.Task.PAUSED_STATUS,
  );
  const canStop = selectedTasks.some(
    ({
      state: {
        details: { status },
      },
    }) => status === Common.Task.RUNNING_STATUS,
  );
  const someSelected = selectedTasks.length > 0;
  const isSingle = selectedTasks.length === 1;

  const addStatus = Status.useAdder();
  const placeLayout = Layout.usePlacer();
  const copyLinkToClipboard = Cluster.useCopyLinkToClipboard();

  const handleEdit = useCallback(
    (key: task.Key) => {
      const task = tasks.find((t) => t.key === key);
      if (task == null) {
        addStatus({
          variant: "error",
          message: "Failed to open task details",
          description: `Task with key ${key} not found`,
        });
        return;
      }
      const layout = createLayout(task);
      placeLayout(layout);
    },
    [tasks, addStatus, placeLayout],
  );
  const handleLink = useCallback(
    (key: task.Key) => {
      const name = tasks.find((t) => t.key === key)?.name;
      if (name == null) {
        addStatus({
          variant: "error",
          message: "Failed to copy link",
          description: `Task with key ${key} not found`,
        });
        return;
      }
      copyLinkToClipboard({ name, ontologyID: task.ontologyID(key) });
    },
    [tasks, addStatus, copyLinkToClipboard],
  );
  const handleChange = useMemo<PMenu.MenuProps["onChange"]>(
    () => ({
      start: () => onStart(keys),
      stop: () => onStop(keys),
      edit: () => handleEdit(keys[0]),
      rename: () => Text.edit(`text-${keys[0]}`),
      link: () => handleLink(keys[0]),
      delete: () => onDelete(keys),
    }),
    [onStart, onStop, handleEdit, handleLink, onDelete, keys],
  );
  return (
    <PMenu.Menu level="small" iconSpacing="small" onChange={handleChange}>
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
          <Link.CopyMenuItem />
          <PMenu.Divider />
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
};
