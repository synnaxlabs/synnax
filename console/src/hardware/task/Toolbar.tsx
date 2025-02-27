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
  Header,
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
import { useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Menu } from "@/components/menu";
import { CSS } from "@/css";
import { NULL_CLIENT_ERROR } from "@/errors";
import { Common } from "@/hardware/common";
import { createLayout } from "@/hardware/task/layouts";
import { SELECTOR_LAYOUT } from "@/hardware/task/Selector";
import { getIcon, parseType } from "@/hardware/task/types";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Modals } from "@/modals";

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  const handleClick = () => {
    placeLayout(SELECTOR_LAYOUT);
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

interface RenameArgs {
  name: string;
  key: task.Key;
}

type Command = "start" | "stop";
interface StartStopArgs {
  command: Command;
  keys: task.Key[];
}

const parseVariant = (variant?: string): Status.Variant | undefined =>
  Status.variantZ.safeParse(variant).data ?? undefined;

const Content = () => {
  const client = Synnax.use();
  const [tasks, setTasks] = useState<task.Task[]>([]);
  const [selected, setSelected] = useState<task.Key[]>([]);
  const handleException = Status.useExceptionHandler();
  const handleRename = useMutation({
    onMutate: ({ name, key }: RenameArgs) => {
      const taskIndex = tasks.findIndex((t) => t.key === key);
      if (taskIndex === -1) return "task";
      setTasks((prev) =>
        prev.map((t, i) => {
          if (i === taskIndex) t.name = name;
          return t;
        }),
      );
      return tasks[taskIndex].name;
    },
    mutationFn: async ({ name, key }) => {
      const tsk = tasks.find((t) => t.key === key);
      if (tsk == null) throw new UnexpectedError(`Task with key ${key} not found`);
      if (states[key].status === "running") {
        const confirmed = await confirm({
          message: `Are you sure you want to rename ${tsk.name} to ${name}?`,
          description: `This will cause ${tsk.name} to stop and be reconfigured.`,
          cancel: { label: "Cancel" },
          confirm: { label: "Rename", variant: "error" },
        });
        if (!confirmed) return;
      }
      if (client == null) throw NULL_CLIENT_ERROR;
      await client.hardware.tasks.create({ ...tsk, name });
    },
    onError: (e, { name, key }, oldName) => {
      if (oldName != null)
        setTasks((prev) =>
          prev.map((t) => {
            if (t.key === key) t.name = oldName;
            return t;
          }),
        );
      if (errors.CANCELED.matches(e)) return;
      handleException(e, `Failed to rename ${oldName ?? "task"} to ${name}`);
    },
  }).mutate;
  const [states, setStates] = useState<Record<task.Key, Common.Task.State>>({});
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
    setTasks(shownTasks);
    const startingStates: Record<task.Key, Common.Task.State> = {};
    shownTasks.forEach(({ key, state }) => {
      startingStates[key] = {
        status: state?.details?.running ? "running" : "paused",
        variant: parseVariant(state?.variant),
      };
    });
    setStates(startingStates);
  }, [client?.key]);
  Observe.useListener({
    key: [client?.key],
    open: async () => client?.hardware.tasks.openStateObserver(),
    onChange: (state) => {
      const key = state.task;
      setTasks((prev) => {
        const task = prev.find((t) => t.key === key);
        if (task == null) return prev;
        task.state = state;
        return [...prev];
      });
      setStates((prev) => {
        const prevTsk = prev[key];
        if (prevTsk == null) return prev;
        const next = { ...prev };
        next[key] = {
          status: state.details?.running ? "running" : "paused",
          variant: parseVariant(state.variant),
        };
        return next;
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
      client.hardware.tasks
        .retrieve(addedOrUpdated)
        .then((changedTasks) => {
          setTasks((prev) => {
            const changedTasksRecord: Record<task.Key, task.Task> = {};
            changedTasks.forEach((task) => {
              changedTasksRecord[task.key] = task;
            });
            const next = prev
              .filter(({ key }) => !removed.has(key))
              .map((t) => changedTasksRecord[t.key] ?? t);
            const existingKeys = new Set(next.map(({ key }) => key));
            return [
              ...next,
              ...changedTasks.filter(
                ({ key, internal, snapshot }) =>
                  !internal && !snapshot && !existingKeys.has(key),
              ),
            ];
          });
        })
        .catch((e) => handleException(e, "Failed to update task toolbar"));
    },
  });
  Observe.useListener({
    key: [client?.key],
    open: async () => client?.hardware.tasks.openCommandObserver(),
    onChange: ({ type, task }) => {
      if (type !== "start" && type !== "stop") return;
      const desiredStatus = type === "start" ? "running" : "paused";
      if (states[task] == null) return;
      if (states[task].status === desiredStatus) return;
      setStates((prev) => {
        const next = { ...prev };
        next[task] = Common.Task.LOADING_STATE;
        return next;
      });
    },
  });
  const confirm = Modals.useConfirm();
  const handleDelete = useMutation<void, Error, string[], task.Task[]>({
    mutationFn: async (keys: string[]) => {
      setSelected([]);
      if (keys.length === 0) return;
      if (client == null) throw NULL_CLIENT_ERROR;
      const deletedNames = tasks
        .filter((task) => keys.includes(task.key))
        .map((task) => task.name);
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
      setStates((prev) => {
        const next = { ...prev };
        keys.forEach((k) => delete next[k]);
        return next;
      });
    },
    onError: (e) => {
      if (errors.CANCELED.matches(e)) return;
      handleException(e, "Failed to delete tasks");
    },
  }).mutate;
  const actions = useMemo(
    () => [{ children: <Icon.Add />, onClick: () => placeLayout(SELECTOR_LAYOUT) }],
    [placeLayout],
  );
  const handleStartStop = useMutation({
    mutationFn: async ({ command, keys }: StartStopArgs) => {
      if (client == null) throw NULL_CLIENT_ERROR;
      const filteredKeys = new Set(
        keys.filter((k) => {
          const status = states[k].status;
          if (status === "loading") return false;
          if (command === "start" && status === "running") return false;
          if (command === "stop" && status === "paused") return false;
          return true;
        }),
      );
      setStates((prev) => {
        const next = { ...prev };
        filteredKeys.forEach((key) => {
          next[key] = Common.Task.LOADING_STATE;
        });
        return next;
      });
      const tasksToExecute = tasks.filter(({ key }) => filteredKeys.has(key));
      await Promise.all(
        tasksToExecute.map(async (tsk) => {
          await tsk.executeCommand(command);
        }),
      );
    },
    onError: (e, { command }) => handleException(e, `Failed to ${command} tasks`),
  }).mutate;
  const handleStart = useCallback(
    (keys: string[]) => handleStartStop({ command: "start", keys }),
    [handleStartStop],
  );
  const handleStop = useCallback(
    (keys: string[]) => handleStartStop({ command: "stop", keys }),
    [handleStartStop],
  );
  const contextMenu = useCallback<NonNullable<PMenu.ContextMenuProps["menu"]>>(
    ({ keys }) => (
      <ContextMenu
        keys={keys}
        tasks={tasks}
        desiredStates={states}
        onDelete={handleDelete}
        onStart={handleStart}
        onStop={handleStop}
      />
    ),
    [handleDelete, handleStart, handleStop, tasks, states],
  );
  const handleListItemStopStart = useCallback(
    (command: Command, key: task.Key) => {
      const status = states[key].status;
      if (status === "loading") return;
      if (command === "start" && status === "running") return;
      if (command === "stop" && status === "paused") return;
      setStates((prev) => {
        const next = { ...prev };
        next[key] = Common.Task.LOADING_STATE;
        return next;
      });
    },
    [states],
  );
  const listItem = useCallback<List.ItemRenderProp<string, task.Task>>(
    ({ key, ...p }) => (
      <TaskListItem
        key={key}
        {...p}
        desiredState={states[key]}
        onStopStart={(command) => handleListItemStopStart(command, key)}
        onRename={(name) => handleRename({ name, key })}
      />
    ),
    [handleListItemStopStart, handleRename, states],
  );
  return (
    <PMenu.ContextMenu menu={contextMenu} {...menuProps}>
      <Align.Space empty style={{ height: "100%" }} className={CSS.B("task-toolbar")}>
        <ToolbarHeader>
          <ToolbarTitle icon={<Icon.Task />}>Tasks</ToolbarTitle>
          <Header.Actions>{actions}</Header.Actions>
        </ToolbarHeader>
        <List.List data={tasks} emptyContent={<EmptyContent />}>
          <List.Selector value={selected} onChange={setSelected} replaceOnSingle>
            <List.Core<string, task.Task>>{listItem}</List.Core>
          </List.Selector>
        </List.List>
      </Align.Space>
    </PMenu.ContextMenu>
  );
};

export const TOOLBAR_NAV_DRAWER_ITEM: Layout.NavDrawerItem = {
  key: "task",
  icon: <Icon.Task />,
  content: <Content />,
  tooltip: "Tasks",
  initialSize: 300,
  minSize: 225,
  maxSize: 400,
};

interface TaskListItemProps extends List.ItemProps<string, task.Task> {
  desiredState: Common.Task.State;
  onStopStart: (command: Command) => void;
  onRename: (name: string) => void;
}

const TaskListItem = ({
  desiredState,
  onStopStart,
  onRename,
  entry: tsk,
  ...rest
}: TaskListItemProps) => {
  const icon = getIcon(tsk.type);
  const handleException = Status.useExceptionHandler();
  const isLoading = desiredState.status === "loading";
  const isRunning = desiredState.status === "running";
  const handleClick: NonNullable<Button.IconProps["onClick"]> = useCallback(
    (e) => {
      e.stopPropagation();
      if (isLoading) return;
      const command = isRunning ? "stop" : "start";
      onStopStart(command);
      tsk
        .executeCommand(command)
        .catch((e) => handleException(e, `Failed to ${command} task`));
    },
    [tsk, desiredState, onStopStart, handleException],
  );
  return (
    <List.ItemFrame
      {...rest}
      entry={tsk}
      justify="spaceBetween"
      align="center"
      rightAligned
    >
      <Align.Space
        direction="y"
        size="small"
        grow
        className={CSS.BE("task", "metadata")}
      >
        <Align.Space direction="x" align="center" size="small">
          <Status.Circle
            variant={desiredState.variant}
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
              id={`text-${tsk.key}`}
              level="p"
              value={tsk.name}
              onChange={onRename}
              allowDoubleClick={false}
            />
          </Text.WithIcon>
        </Align.Space>
        <Text.Text level="small" shade={6}>
          {parseType(tsk.type)}
        </Text.Text>
      </Align.Space>
      <Button.Icon
        variant="outlined"
        loading={isLoading}
        onClick={handleClick}
        tooltip={`${isRunning ? "Stop" : "Start"} ${tsk.name}`}
      >
        {isRunning ? <Icon.Pause /> : <Icon.Play />}
      </Button.Icon>
    </List.ItemFrame>
  );
};

interface ContextMenuProps {
  keys: string[];
  tasks: task.Task[];
  desiredStates: Record<task.Key, Common.Task.State>;
  onDelete: (keys: string[]) => void;
  onStart: (keys: string[]) => void;
  onStop: (keys: string[]) => void;
}

const ContextMenu = ({
  keys,
  tasks,
  desiredStates,
  onDelete,
  onStart,
  onStop,
}: ContextMenuProps) => {
  const selectedKeys = new Set(keys);
  const selectedTasks = tasks.filter(({ key }) => selectedKeys.has(key));

  const canStart = selectedTasks.some(
    ({ key }) => desiredStates[key].status === "paused",
  );
  const canStop = selectedTasks.some(
    ({ key }) => desiredStates[key].status === "running",
  );
  const someSelected = selectedTasks.length > 0;
  const isSingle = selectedTasks.length === 1;

  const addStatus = Status.useAdder();
  const placeLayout = Layout.usePlacer();
  const copyLinkToClipboard = Link.useCopyToClipboard();

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
    [tasks, desiredStates, placeLayout],
  );
  const handleLink = useCallback(
    (key: task.Key) => {
      const name = tasks.find((t) => t.key === key)?.name;
      if (name == null) return;
      copyLinkToClipboard({ name, ontologyID: task.ontologyID(key) });
    },
    [tasks, copyLinkToClipboard],
  );
  const handleChange: PMenu.MenuProps["onChange"] = useMemo(
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
