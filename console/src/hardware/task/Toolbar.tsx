// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/task/Toolbar.css";

import { DisconnectedError, task, UnexpectedError } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Icon,
  List,
  Menu as PMenu,
  Status,
  Synnax,
  Task,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { errors, strings, TimeSpan } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Menu, Toolbar } from "@/components";
import { CSS } from "@/css";
import { Export } from "@/export";
import { Common } from "@/hardware/common";
import { createLayout } from "@/hardware/task/layouts";
import { SELECTOR_LAYOUT } from "@/hardware/task/Selector";
import { getIcon, parseType } from "@/hardware/task/types";
import { useRangeSnapshot } from "@/hardware/task/useRangeSnapshot";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Modals } from "@/modals";
import { Range } from "@/range";

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  const handleClick = () => {
    placeLayout(SELECTOR_LAYOUT);
  };
  return (
    <Align.Space empty style={{ height: "100%", position: "relative" }}>
      <Align.Center y style={{ height: "100%" }} gap="small">
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
  command: Common.Task.Command;
  keys: task.Key[];
}

const updateTaskStatus =
  (key: task.Key, update: (prev: task.Status) => task.Status) => (prev: task.Task[]) =>
    prev.map((tsk) => {
      if (tsk.key === key && tsk.status != null) tsk.status = update(tsk.status);
      return tsk;
    });

const Content = () => {
  const client = Synnax.use();
  const [tasks, setTasks] = useState<task.Task[]>([]);
  const [selected, setSelected] = useState<task.Key[]>([]);
  const handleError = Status.useErrorHandler();
  const confirm = Modals.useConfirm();
  const rename = useMutation({
    onMutate: ({ key }) => tasks.find((t) => t.key === key)?.name ?? "task",
    mutationFn: async ({ name, key }: RenameArgs) => {
      const tsk = tasks.find((t) => t.key === key);
      if (tsk == null) throw new UnexpectedError(`Task with key ${key} not found`);
      if (tsk.status?.details.running === true) {
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
      if (client == null) throw new DisconnectedError();
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
  useAsyncEffect(
    async (signal) => {
      if (client == null) {
        setTasks([]);
        return;
      }
      const all = await client.hardware.tasks.list({ includeStatus: true });
      if (signal.aborted) return;
      const shown = all.filter(({ internal, snapshot }) => !internal && !snapshot);
      setTasks(shown);
    },
    [client?.key],
  );
  const handleStatusChange = useCallback((status: task.Status) => {
    setTasks((prev) => {
      const tsk = prev.find((t) => t.key === status.details.task);
      if (tsk == null) return prev;
      tsk.status = status;
      return [...prev];
    });
  }, []);
  Task.useStatusSynchronizer(handleStatusChange);
  const handleSet = useCallback(
    (key: task.Key) => {
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        const tk = await client.hardware.tasks.retrieve({ key, includeStatus: true });
        setTasks((prev) => {
          const existing = prev.find((t) => t.key === tk.key);
          if (existing == null) return [...prev, tk];
          return prev.map((t) => (t.key === tk.key ? tk : t));
        });
      }, "Failed to update task toolbar from sy_task_set channel");
    },
    [client?.key],
  );
  const handleDeletedTasks = useCallback((key: task.Key) => {
    setTasks((prevTasks) => prevTasks.filter((t) => t.key !== key));
    setSelected((prev) => prev.filter((k) => k !== key));
  }, []);
  Task.useSetSynchronizer(handleSet);
  Task.useDeleteSynchronizer(handleDeletedTasks);

  const handleCommandUpdate = useCallback(
    ({ task, type }: task.Command) => {
      const status = tasks.find(({ key }) => key === task)?.status;
      if (status == null) return;
      if (Common.Task.shouldExecuteCommand(status, type))
        setTasks((prev) =>
          prev.map((tsk) => {
            if (tsk.key === task && tsk.status != null) tsk.status.variant = "loading";
            return tsk;
          }),
        );
    },
    [tasks],
  );
  Task.useCommandSynchronizer(handleCommandUpdate);
  const handleDelete = useMutation({
    mutationFn: async (keys: string[]) => {
      setSelected([]);
      if (keys.length === 0) return;
      if (client == null) throw new DisconnectedError();
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
      if (errors.Canceled.matches(e)) return;
      handleError(e, "Failed to delete tasks");
    },
  }).mutate;
  const actions = useMemo(
    () => [{ children: <Icon.Add />, onClick: () => placeLayout(SELECTOR_LAYOUT) }],
    [placeLayout],
  );
  const startOrStop = useMutation({
    mutationFn: async ({ command, keys }: StartStopArgs) => {
      if (client == null) throw new DisconnectedError();
      const filteredKeys = new Set(
        keys.filter((k) => {
          const status = tasks.find(({ key }) => key === k)?.status;
          if (status == null) throw new UnexpectedError(`Task with key ${k} not found`);
          return Common.Task.shouldExecuteCommand(status, command);
        }),
      );
      setTasks((prev) =>
        prev.map((tsk) => {
          if (filteredKeys.has(tsk.key) && tsk.status != null)
            tsk.status.variant = "loading";
          return tsk;
        }),
      );
      const tasksToExecute = tasks.filter(({ key }) => filteredKeys.has(key));
      tasksToExecute.forEach((t) => {
        t.executeCommandSync(command, TimeSpan.fromSeconds(10), {}).catch((e) => {
          setTasks(
            updateTaskStatus(t.key, (prev) => ({
              ...prev,
              variant: "error",
              message: e.message,
            })),
          );
        });
      });
    },
    onError: (e, { command }) => handleError(e, `Failed to ${command} tasks`),
  }).mutate;
  const handleStart = useCallback(
    (keys: string[]) => startOrStop({ command: "start", keys }),
    [startOrStop],
  );
  const handleStop = useCallback(
    (keys: string[]) => startOrStop({ command: "stop", keys }),
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
    (command: Common.Task.Command, key: task.Key) =>
      startOrStop({ command, keys: [key] }),
    [startOrStop],
  );
  const listItem = useCallback<List.ItemRenderProp<string, task.Task>>(
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
    <PMenu.ContextMenu menu={contextMenu} {...menuProps}>
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
            <List.Core<task.Key, task.Task>>{listItem}</List.Core>
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
  trigger: ["T"],
  tooltip: "Tasks",
  initialSize: 300,
  minSize: 225,
  maxSize: 400,
};

interface TaskListItemProps extends List.ItemProps<task.Key, task.Task> {
  onStopStart: (command: Common.Task.Command) => void;
  onRename: (name: string) => void;
}

const TaskListItem = ({ onStopStart, onRename, ...rest }: TaskListItemProps) => {
  const { key, name, status, type } = rest.entry;
  const details = status?.details;
  let variant = status?.variant;
  const icon = getIcon(type);
  const isLoading = variant === "loading";
  const isRunning = details?.running === true;
  if (!isRunning && variant === "success") variant = "info";
  const handleClick = useCallback<NonNullable<Button.IconProps["onClick"]>>(
    (e) => {
      e.stopPropagation();
      const command = isRunning ? "stop" : "start";
      onStopStart(command);
    },
    [isRunning, onStopStart],
  );
  return (
    <List.ItemFrame {...rest} justify="spaceBetween" align="center">
      <Align.Space y gap="small" grow className={CSS.BE("task", "metadata")}>
        <Align.Space x align="center" gap="small">
          <Status.Indicator
            variant={variant}
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
  tasks: task.Task[];
}

const ContextMenu = ({ keys, tasks, onDelete, onStart, onStop }: ContextMenuProps) => {
  const selectedKeys = new Set(keys);
  const selectedTasks = tasks.filter(({ key }) => selectedKeys.has(key));
  const activeRange = Range.useSelect();
  const snapshotToActiveRange = useRangeSnapshot();

  const canStart = selectedTasks.some(
    ({ status }) => status?.details.running === false,
  );
  const canStop = selectedTasks.some(({ status }) => status?.details.running === true);
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
  const handleExport = Common.Task.useExport();
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
      export: () => handleExport(keys[0]),
      delete: () => onDelete(keys),
      rangeSnapshot: () =>
        snapshotToActiveRange(
          selectedTasks.map(({ name, ontologyID }) => ({ id: ontologyID, name })),
        ),
    }),
    [
      onStart,
      onStop,
      handleEdit,
      handleLink,
      onDelete,
      keys,
      snapshotToActiveRange,
      selectedTasks,
    ],
  );
  const showSnapshotToActiveRange =
    activeRange?.persisted === true && selectedTasks.length > 0;
  return (
    <PMenu.Menu level="small" gap="small" onChange={handleChange}>
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
          <Export.MenuItem />
          <Link.CopyMenuItem />
          <PMenu.Divider />
        </>
      )}
      {showSnapshotToActiveRange && (
        <>
          <Range.SnapshotMenuItem range={activeRange} key="snapshot" />
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
