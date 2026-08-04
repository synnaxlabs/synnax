// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/task/Toolbar.css";

import { task } from "@synnaxlabs/client";
import {
  Access,
  Button,
  Flex,
  type Flux,
  Icon,
  List,
  Menu,
  Select,
  Status,
  stopPropagation,
  Synnax,
  Task,
  Text,
  Tooltip,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

import { useOpenSelector } from "@/feature/task/Selector";
import { useRangeSnapshot } from "@/feature/task/useRangeSnapshot";
import { useSetDataSaving } from "@/feature/task/useSetDataSaving";
import { Cluster } from "@/platform/cluster";
import { ContextMenu as PlatformContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Export } from "@/platform/export";
import { Link } from "@/platform/link";
import { Modals } from "@/platform/modals";
import { type Nav } from "@/platform/nav";
import { Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { Task as PlatformTask } from "@/platform/task";
import { Toolbar } from "@/platform/toolbar";
import { Session } from "@/session";

const EmptyContent = () => {
  const openSelector = useOpenSelector();
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);
  return (
    <Empty.Action
      message="No existing tasks."
      action={hasCreatePermission ? "Create a task" : undefined}
      onClick={() => openSelector()}
    />
  );
};

const INITIAL_QUERY: Task.ListQuery = {
  internal: false,
  snapshot: false,
};

const filter = (task: task.Task) =>
  !task.internal && !task.snapshot && task.type !== "arc";

const Content = () => {
  const client = Synnax.use();
  const [selected, setSelected] = useState<task.Key[]>([]);
  const addStatus = Status.useAdder();
  const confirmDelete = Modals.useConfirmDelete({ type: "Task" });
  const menuProps = Menu.useContextMenu();
  const openTab = Panel.useOpenTab();
  const openSelector = useOpenSelector();
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);
  const { data, getItem, subscribe, retrieve } = Task.useList({
    initialQuery: INITIAL_QUERY,
    filter,
  });
  const { fetchMore } = List.usePager({ retrieve, pageSize: 1e3 });

  const { update: rename } = Task.useRename();

  const { update: handleDelete } = Task.useDelete({
    beforeUpdate: useCallback(
      async ({ data: keys }: Flux.BeforeUpdateParams<Task.DeleteParams>) => {
        setSelected([]);
        if (keys.length === 0) return false;
        if (!(await confirmDelete(getItem(array.toArray(keys))))) return false;
        return keys;
      },
      [client, getItem, confirmDelete],
    ),
    afterFailure: ({ status }) => addStatus(status),
  });

  const { update: runCommand } = Task.useCommand();
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
  const { update: setDataSaving } = useSetDataSaving();
  const handleEnableDataSaving = useCallback(
    (keys: task.Key[]) =>
      keys.forEach((key) => setDataSaving({ key, dataSaving: true })),
    [setDataSaving],
  );
  const handleDisableDataSaving = useCallback(
    (keys: task.Key[]) =>
      keys.forEach((key) => setDataSaving({ key, dataSaving: false })),
    [setDataSaving],
  );
  const handleEdit = useCallback(
    (key: task.Key) => openTab({ variant: "resource", resource: task.ontologyID(key) }),
    [openTab],
  );
  const contextMenu = useCallback<NonNullable<Menu.ContextMenuProps["menu"]>>(
    ({ keys }) => (
      <ContextMenu
        keys={keys}
        tasks={getItem(keys)}
        onDelete={handleDelete}
        onStart={handleStart}
        onStop={handleStop}
        onEdit={handleEdit}
        onEnableDataSaving={handleEnableDataSaving}
        onDisableDataSaving={handleDisableDataSaving}
      />
    ),
    [
      handleDelete,
      handleStart,
      handleStop,
      handleEnableDataSaving,
      handleDisableDataSaving,
    ],
  );
  const handleListItemStopStart = useCallback(
    (command: PlatformTask.Command, key: task.Key) => handleCommand([key], command),
    [handleCommand],
  );
  return (
    <Menu.ContextMenu menu={contextMenu} {...menuProps}>
      <Toolbar.Content className={CSS(CSS.B("task-toolbar"), menuProps.className)}>
        <Toolbar.Header>
          <Toolbar.Title icon={<Icon.Task />}>Tasks</Toolbar.Title>
          {hasCreatePermission && (
            <Toolbar.Actions>
              <Toolbar.Action
                tooltip="Create task"
                onClick={() => openSelector()}
                variant="filled"
              >
                <Icon.Add />
              </Toolbar.Action>
            </Toolbar.Actions>
          )}
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
    </Menu.ContextMenu>
  );
};

export const TOOLBAR: Nav.Toolbar = {
  key: "task",
  icon: <Icon.Task />,
  content: <Content />,
  trigger: ["T"],
  tooltip: "Tasks",
  initialSize: 300,
  sizeBounds: { lower: 225, upper: 400 },
  useVisible: () => Access.useRetrieveGranted(task.TYPE_ONTOLOGY_ID),
};

interface TaskListItemProps extends List.ItemProps<task.Key> {
  onStopStart: (command: PlatformTask.Command) => void;
  onRename: (name: string) => void;
}

const TaskListItem = ({ onStopStart, onRename, ...rest }: TaskListItemProps) => {
  const { itemKey } = rest;
  const { getIcon, parseType } = PlatformTask.useRegistry();
  const task_ = List.useItem<task.Key, task.Task>(itemKey);
  const hasUpdatePermission = Access.useUpdateGranted(task.ontologyID(itemKey));
  const details = task_?.status?.details;
  let variant = task_?.status?.variant;
  const icon = getIcon(task_?.type ?? "");
  const isLoading = variant === "loading";
  const isRunning = details?.running === true;
  const isDrifted = task_ != null && task.drifted(task_.payload);
  if (!isRunning && variant === "success") variant = "info";
  const handleStartStopClick = useCallback(
    () => onStopStart(isRunning ? "stop" : "start"),
    [isRunning, onStopStart],
  );
  return (
    <Select.ListItem {...rest} justify="between" align="center">
      <Flex.Box y gap="small" grow className={CSS.BE("task", "metadata")}>
        <Flex.Box x align="center" gap="small">
          <Status.Indicator
            variant={variant}
            className={CSS.BE("task", "status-indicator")}
          />
          <Flex.Box x className={CSS.BE("task", "title")} align="center">
            {icon}
            <Text.MaybeEditable
              id={`text-${itemKey}`}
              value={task_?.name ?? ""}
              onChange={hasUpdatePermission ? onRename : undefined}
              allowDoubleClick={false}
              overflow="ellipsis"
              weight={500}
            />
            {isDrifted && (
              <Tooltip.Dialog>
                <Text.Text level="small">Configuration changed since deploy</Text.Text>
                <Text.Text
                  className={CSS.BE("task", "drift-indicator")}
                  level="small"
                  status="warning"
                >
                  <Icon.Warning />
                </Text.Text>
              </Tooltip.Dialog>
            )}
          </Flex.Box>
        </Flex.Box>
        <Text.Text level="small" color={10}>
          {parseType(task_?.type ?? "")}
        </Text.Text>
      </Flex.Box>
      {hasUpdatePermission && (
        <Button.Button
          variant="outlined"
          status={isLoading ? "loading" : undefined}
          onClick={handleStartStopClick}
          onDoubleClick={stopPropagation}
          tooltip={`${isRunning ? "Stop" : "Start"} ${task_?.name ?? ""}`}
        >
          {isRunning ? <Icon.Pause /> : <Icon.Play />}
        </Button.Button>
      )}
    </Select.ListItem>
  );
};

interface ContextMenuProps {
  keys: task.Key[];
  onDelete: (keys: task.Key[]) => void;
  onStart: (keys: task.Key[]) => void;
  onStop: (keys: task.Key[]) => void;
  onEdit: (key: task.Key) => void;
  onEnableDataSaving: (keys: task.Key[]) => void;
  onDisableDataSaving: (keys: task.Key[]) => void;
  tasks: task.Task[];
}

const ContextMenu = ({
  keys,
  tasks: selectedTasks,
  onDelete,
  onStart,
  onStop,
  onEdit,
  onEnableDataSaving,
  onDisableDataSaving,
}: ContextMenuProps) => {
  const activeRange = Session.Range.useSelectState();
  const snapshotToActiveRange = useRangeSnapshot();
  const ontologyIDs = task.ontologyID(keys);
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);
  const hasDeletePermission = Access.useDeleteGranted(ontologyIDs);
  const hasUpdatePermission = Access.useUpdateGranted(ontologyIDs);

  const canStart = selectedTasks.some(
    ({ status }) => status?.details.running === false,
  );
  const canStop = selectedTasks.some(({ status }) => status?.details.running === true);
  const someSelected = selectedTasks.length > 0;
  const isSingle = selectedTasks.length === 1;

  // Only tasks with a dataSaving field in their config (primarily read tasks) are
  // eligible for these menu items. Write tasks without this field are excluded.
  const dataSavingTasks = selectedTasks.filter(
    ({ config }) =>
      config != null && typeof config === "object" && "dataSaving" in config,
  );
  const canEnableDataSaving = dataSavingTasks.some(
    ({ config }) =>
      config != null &&
      typeof config === "object" &&
      "dataSaving" in config &&
      config.dataSaving === false,
  );
  const canDisableDataSaving = dataSavingTasks.some(
    ({ config }) =>
      config != null &&
      typeof config === "object" &&
      "dataSaving" in config &&
      config.dataSaving === true,
  );

  const addStatus = Status.useAdder();
  const copyLinkToClipboard = Cluster.useCopyLinkToClipboard();

  const handleExport = Export.use();
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
  const showSnapshotToActiveRange =
    activeRange?.persisted === true && selectedTasks.length > 0;
  return (
    <PlatformContextMenu.Menu>
      {hasUpdatePermission && (
        <>
          {canStart && (
            <Menu.Item itemKey="start" onClick={() => onStart(keys)}>
              <Icon.Play />
              Start
            </Menu.Item>
          )}
          {canStop && (
            <Menu.Item itemKey="stop" onClick={() => onStop(keys)}>
              <Icon.Pause />
              Stop
            </Menu.Item>
          )}
        </>
      )}
      <Menu.Divider />
      {isSingle && (
        <Menu.Item itemKey="edit" onClick={() => onEdit(keys[0])}>
          <Icon.Edit />
          Edit configuration
        </Menu.Item>
      )}
      {hasUpdatePermission && isSingle && (
        <PlatformContextMenu.RenameItem onClick={() => Text.edit(`text-${keys[0]}`)} />
      )}
      <Menu.Divider />
      {hasUpdatePermission && (
        <>
          {canEnableDataSaving && (
            <Menu.Item
              itemKey="enableDataSaving"
              onClick={() => onEnableDataSaving(keys)}
            >
              <Icon.Save />
              Enable data saving
            </Menu.Item>
          )}
          {canDisableDataSaving && (
            <Menu.Item
              itemKey="disableDataSaving"
              onClick={() => onDisableDataSaving(keys)}
            >
              <Icon.Disable />
              Disable data saving
            </Menu.Item>
          )}
        </>
      )}
      {hasCreatePermission && showSnapshotToActiveRange && (
        <Range.SnapshotMenuItem
          range={activeRange}
          key="snapshot"
          onClick={() =>
            snapshotToActiveRange({
              tasks: selectedTasks.map(({ name, ontologyID: { key } }) => ({
                key,
                name,
              })),
            })
          }
        />
      )}
      <Menu.Divider />
      {isSingle && (
        <>
          <Export.ContextMenuItem
            onClick={() => handleExport(task.ontologyID(keys[0]))}
          />
          <Link.CopyContextMenuItem onClick={() => handleLink(keys[0])} />
        </>
      )}
      <Menu.Divider />
      {hasDeletePermission && someSelected && (
        <PlatformContextMenu.DeleteItem onClick={() => onDelete(keys)} />
      )}
      <Menu.Divider />
      <PlatformContextMenu.ReloadConsoleItem />
    </PlatformContextMenu.Menu>
  );
};
