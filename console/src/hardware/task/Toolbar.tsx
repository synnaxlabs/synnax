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
import {
  Button,
  Flex,
  type Flux,
  Icon,
  List,
  Menu as PMenu,
  Select,
  Status,
  stopPropagation,
  Synnax,
  Task,
  Text,
} from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";
import { useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { EmptyAction, Menu, Toolbar } from "@/components";
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
  const handleClick = () => placeLayout(SELECTOR_LAYOUT);
  return (
    <EmptyAction
      message="No existing tasks."
      action="Create a task"
      onClick={handleClick}
    />
  );
};

const INITIAL_QUERY: Task.ListQuery = {
  internal: false,
  snapshot: false,
};

const filter = (task: task.Task) => !task.internal && !task.snapshot;

const Content = () => {
  const client = Synnax.use();
  const [selected, setSelected] = useState<task.Key[]>([]);
  const addStatus = Status.useAdder();
  const confirm = Modals.useConfirm();
  const menuProps = PMenu.useContextMenu();
  const dispatch = useDispatch();
  const placeLayout = Layout.usePlacer();
  const { data, getItem, subscribe, retrieve } = Task.useList({
    initialQuery: INITIAL_QUERY,
    filter,
  });
  const { fetchMore } = List.usePager({ retrieve, pageSize: 1e3 });

  const { update: rename } = Task.useRename({
    beforeUpdate: useCallback(
      async ({ data, rollbacks }: Flux.BeforeUpdateParams<Task.UseRenameArgs>) => {
        const { key, name } = data;
        const tsk = getItem(key);
        if (tsk == null) throw new UnexpectedError(`Task with key ${key} not found`);
        const oldName = tsk.name;
        if (tsk.status?.details.running === true) {
          const confirmed = await confirm({
            message: `Are you sure you want to rename ${tsk.name} to ${name}?`,
            description: `This will cause ${tsk.name} to stop and be reconfigured.`,
            cancel: { label: "Cancel" },
            confirm: { label: "Rename", variant: "error" },
          });
          if (!confirmed) return false;
        }
        dispatch(Layout.rename({ key, name }));
        rollbacks.push(() => dispatch(Layout.rename({ key, name: oldName })));
        return data;
      },
      [],
    ),
  });

  const { update: handleDelete } = Task.useDelete({
    beforeUpdate: useCallback(
      async ({ data: keys }: Flux.BeforeUpdateParams<Task.DeleteParams>) => {
        setSelected([]);
        if (keys.length === 0) return false;
        const names = strings.naturalLanguageJoin(
          getItem(array.toArray(keys)).map(({ name }) => name),
          "tasks",
        );
        const confirmed = await confirm({
          message: `Are you sure you want to delete ${names}?`,
          description: "This action cannot be undone.",
          cancel: { label: "Cancel" },
          confirm: { label: "Delete", variant: "error" },
        });
        if (!confirmed) return false;
        dispatch(Layout.remove({ keys: array.toArray(keys) }));
        return keys;
      },
      [client, dispatch, getItem],
    ),
    afterFailure: ({ status }) => addStatus(status),
  });

  const handleCommandStatus = useCallback(
    ({ data: statuses }: Flux.AfterSuccessParams<task.Status[]>) =>
      statuses.forEach((status) => addStatus(status)),
    [addStatus],
  );

  const { update: runCommand } = Task.useCommand({
    afterSuccess: handleCommandStatus,
    afterFailure: useCallback(
      ({ status }: Flux.AfterFailureParams<Task.CommandParams>) => addStatus(status),
      [addStatus],
    ),
  });
  const handleCommand = useCallback(
    (keys: string[], type: string) => runCommand(keys.map((k) => ({ task: k, type }))),
    [runCommand],
  );
  const handleStart = useCallback(
    (keys: string[]) => handleCommand(keys, "start"),
    [handleCommand],
  );
  const handleStop = useCallback(
    (keys: string[]) => handleCommand(keys, "stop"),
    [handleCommand],
  );
  const handleEdit = useCallback(
    (key: task.Key) => {
      const task = getItem(key);
      if (task == null)
        return addStatus({
          variant: "error",
          message: "Failed to open task details",
          description: `Task with key ${key} not found`,
        });
      const layout = createLayout(task);
      placeLayout(layout);
    },
    [selected, addStatus, placeLayout],
  );
  const contextMenu = useCallback<NonNullable<PMenu.ContextMenuProps["menu"]>>(
    ({ keys }) => (
      <ContextMenu
        keys={keys}
        tasks={getItem(keys)}
        onDelete={handleDelete}
        onStart={handleStart}
        onStop={handleStop}
        onEdit={handleEdit}
      />
    ),
    [handleDelete, handleStart, handleStop],
  );
  const handleListItemStopStart = useCallback(
    (command: Common.Task.Command, key: task.Key) => handleCommand([key], command),
    [handleCommand],
  );
  return (
    <PMenu.ContextMenu menu={contextMenu} {...menuProps}>
      <Toolbar.Content className={CSS(CSS.B("task-toolbar"), menuProps.className)}>
        <Toolbar.Header padded>
          <Toolbar.Title icon={<Icon.Task />}>Tasks</Toolbar.Title>
          <Toolbar.Actions>
            <Toolbar.Action onClick={() => placeLayout(SELECTOR_LAYOUT)}>
              <Icon.Add />
            </Toolbar.Action>
          </Toolbar.Actions>
        </Toolbar.Header>
        <Select.Frame
          multiple
          data={data}
          getItem={getItem}
          subscribe={subscribe}
          value={selected}
          onChange={setSelected}
          onFetchMore={fetchMore}
          replaceOnSingle
        >
          <List.Items<task.Key, task.Task>
            full="y"
            emptyContent={<EmptyContent />}
            onContextMenu={menuProps.open}
          >
            {({ key, ...p }) => (
              <TaskListItem
                key={key}
                {...p}
                onStopStart={(command) => handleListItemStopStart(command, key)}
                onRename={(name) => rename({ name, key })}
                onDoubleClick={() => handleEdit(key)}
              />
            )}
          </List.Items>
        </Select.Frame>
      </Toolbar.Content>
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

interface TaskListItemProps extends List.ItemProps<task.Key> {
  onStopStart: (command: Common.Task.Command) => void;
  onRename: (name: string) => void;
}

const TaskListItem = ({ onStopStart, onRename, ...rest }: TaskListItemProps) => {
  const { itemKey } = rest;
  const task = List.useItem<task.Key, task.Task>(itemKey);
  const details = task?.status?.details;
  let variant = task?.status?.variant;
  const icon = getIcon(task?.type ?? "");
  const isLoading = variant === "loading";
  const isRunning = details?.running === true;
  if (!isRunning && variant === "success") variant = "info";
  const handleStartStopClick = useCallback<NonNullable<Button.ButtonProps["onClick"]>>(
    (e) => {
      e.stopPropagation();
      const command = isRunning ? "stop" : "start";
      onStopStart(command);
    },
    [isRunning, onStopStart],
  );
  return (
    <Select.ListItem {...rest} justify="between" align="center">
      <Flex.Box y gap="small" grow className={CSS.BE("task", "metadata")}>
        <Flex.Box x align="center" gap="small">
          <Status.Indicator
            variant={variant}
            style={{ fontSize: "2rem", minWidth: "2rem" }}
          />
          <Flex.Box x className={CSS.BE("task", "title")} align="center">
            {icon}
            <Text.MaybeEditable
              id={`text-${itemKey}`}
              value={task?.name ?? ""}
              onChange={onRename}
              allowDoubleClick={false}
              overflow="ellipsis"
              weight={500}
            />
          </Flex.Box>
        </Flex.Box>
        <Text.Text level="small" color={10}>
          {parseType(task?.type ?? "")}
        </Text.Text>
      </Flex.Box>
      <Button.Button
        variant="outlined"
        status={isLoading ? "loading" : undefined}
        onClick={handleStartStopClick}
        onDoubleClick={stopPropagation}
        tooltip={`${isRunning ? "Stop" : "Start"} ${task?.name ?? ""}`}
      >
        {isRunning ? <Icon.Pause /> : <Icon.Play />}
      </Button.Button>
    </Select.ListItem>
  );
};

interface ContextMenuProps {
  keys: task.Key[];
  onDelete: (keys: task.Key[]) => void;
  onStart: (keys: task.Key[]) => void;
  onStop: (keys: task.Key[]) => void;
  onEdit: (key: task.Key) => void;
  tasks: task.Task[];
}

const ContextMenu = ({
  keys,
  tasks: selectedTasks,
  onDelete,
  onStart,
  onStop,
  onEdit,
}: ContextMenuProps) => {
  const activeRange = Range.useSelect();
  const snapshotToActiveRange = useRangeSnapshot();

  const canStart = selectedTasks.some(
    ({ status }) => status?.details.running === false,
  );
  const canStop = selectedTasks.some(({ status }) => status?.details.running === true);
  const someSelected = selectedTasks.length > 0;
  const isSingle = selectedTasks.length === 1;

  const addStatus = Status.useAdder();
  const copyLinkToClipboard = Cluster.useCopyLinkToClipboard();

  const handleExport = Common.Task.useExport();
  const handleLink = useCallback(
    (key: task.Key) => {
      const name = selectedTasks.find((t) => t.key === key)?.name;
      if (name == null)
        return addStatus({
          variant: "error",
          message: "Failed to copy link",
          description: `Task with key ${key} not found`,
        });
      copyLinkToClipboard({ name, ontologyID: task.ontologyID(key) });
    },
    [selectedTasks, addStatus, copyLinkToClipboard],
  );
  const handleChange = useMemo<PMenu.MenuProps["onChange"]>(
    () => ({
      start: () => onStart(keys),
      stop: () => onStop(keys),
      edit: () => onEdit(keys[0]),
      rename: () => Text.edit(`text-${keys[0]}`),
      link: () => handleLink(keys[0]),
      export: () => handleExport(keys[0]),
      delete: () => onDelete(keys),
      rangeSnapshot: () =>
        snapshotToActiveRange({
          tasks: selectedTasks.map(({ name, ontologyID: { key } }) => ({ key, name })),
        }),
    }),
    [
      onStart,
      onStop,
      onEdit,
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
        <PMenu.Item itemKey="start">
          <Icon.Play />
          Start
        </PMenu.Item>
      )}
      {canStop && (
        <PMenu.Item itemKey="stop">
          <Icon.Pause />
          Stop
        </PMenu.Item>
      )}
      {(canStart || canStop) && <PMenu.Divider />}
      {isSingle && (
        <>
          <PMenu.Item itemKey="edit">
            <Icon.Edit />
            Edit configuration
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
          <PMenu.Item itemKey="delete">
            <Icon.Delete />
            Delete
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
      <Menu.ReloadConsoleItem />
    </PMenu.Menu>
  );
};
