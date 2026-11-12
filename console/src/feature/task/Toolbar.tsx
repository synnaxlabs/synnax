// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/task/Toolbar.css";

import { arc, query, task } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Store } from "@synnaxlabs/lyra/store";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { stopPropagation } from "@synnaxlabs/lyra/util";
import { Access, Arc, type Flux, Synnax, Task } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { useState } from "react";

import { ArcItem } from "@/feature/task/ArcItem";
import { useOpenSelector } from "@/feature/task/Selector";
import { useRangeSnapshot } from "@/feature/task/useRangeSnapshot";
import {
  type SetDataSavingParams,
  useSetDataSaving,
} from "@/feature/task/useSetDataSaving";
import { Analytics } from "@/platform/analytics";
import { Arc as PlatformArc } from "@/platform/arc";
import { ContextMenu as PlatformContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Export } from "@/platform/export";
import { Framer } from "@/platform/framer";
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
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_RESOURCE_ID);
  return (
    <Empty.Action
      message="No tasks"
      action={hasCreatePermission ? "Create task" : undefined}
      onClick={() => openSelector()}
    />
  );
};

const INITIAL_QUERY: Task.ListQuery = {
  where: { internal: { eq: false }, snapshot: { eq: false } },
};

const filter = (task: task.Task) => !task.internal && !task.snapshot;

const Content = () => {
  const [selected, setSelected] = useState<task.Key[]>([]);
  const confirmDelete = Modals.useConfirmDelete({ type: "Task" });
  const menuProps = Menu.useContextMenu();
  const openTab = Panel.useOpenTab();
  const openSelector = useOpenSelector();
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_RESOURCE_ID);
  const { data, getItem, subscribe, more, answered } = Task.useList({
    query: INITIAL_QUERY,
    filter,
    pageSize: 1000,
  });

  const { update: rename } = Task.useRename();

  const dispatch = Session.useDispatch();
  const { update: deleteArcs } = Arc.useDelete({
    beforeUpdate: async ({ data }: Flux.BeforeUpdateParams<arc.Key | arc.Key[]>) => {
      dispatch(Session.Arc.remove({ keys: array.toArray(data) }));
      return data;
    },
  });

  // The row of an Arc deletes the Arc, as the menu of the Arc does.
  const { update: handleDelete } = Task.useDelete({
    beforeUpdate: async ({
      data: keys,
    }: Flux.BeforeUpdateParams<Task.DeleteParams>) => {
      setSelected([]);
      if (keys.length === 0) return false;
      const rows = getItem(array.toArray(keys));
      if (!(await confirmDelete(rows))) return false;
      const arcKeys = rows.map(arc.task.arcKeyOf).filter((key) => key != null);
      if (arcKeys.length > 0) deleteArcs(arcKeys);
      const taskKeys = rows
        .filter(({ type }) => type !== arc.task.TYPE)
        .map(({ key }) => key);
      return taskKeys.length > 0 ? taskKeys : false;
    },
  });

  const { capture } = Analytics.use();
  const { update: runCommand } = Task.useCommand({
    afterSuccess: ({ data }: Flux.AfterSuccessParams<Task.CommandParams>) =>
      getItem(
        array
          .toArray(data)
          .filter(({ type }) => type === "start")
          .map(({ task }) => task),
      ).forEach(({ type }) => capture("task_started", { type })),
  });
  const captureDeploy = PlatformArc.useCaptureDeploy();
  const { update: runArcCommand } = Task.useCommand({ afterSuccess: captureDeploy });
  const handleCommand = (keys: task.Key[], type: string) => {
    const rows = getItem(keys);
    const commandsOf = (tasks: task.Task[]) =>
      tasks.map(({ key }) => ({ task: key, type }));
    const arcTasks = rows.filter((t) => t.type === arc.task.TYPE);
    const otherTasks = rows.filter((t) => t.type !== arc.task.TYPE);
    if (otherTasks.length > 0) runCommand(commandsOf(otherTasks));
    if (arcTasks.length > 0) runArcCommand(commandsOf(arcTasks));
  };
  const handleStart = (keys: string[]) => handleCommand(keys, "start");
  const handleStop = (keys: string[]) => handleCommand(keys, "stop");
  const { update: setDataSaving } = useSetDataSaving();
  const handleEnableDataSaving = (keys: task.Key[]) =>
    keys.forEach((key) => setDataSaving({ key, dataSaving: true }));
  const handleDisableDataSaving = (keys: task.Key[]) =>
    keys.forEach((key) => setDataSaving({ key, dataSaving: false }));
  const handleEdit = (key: task.Key) =>
    openTab({ variant: "resource", resource: task.resourceID(key) });
  const { update: renameArc } = Arc.useRename();
  const handleEditArc = (key: arc.Key) =>
    openTab({ variant: "resource", resource: arc.resourceID(key) });
  const getArc = useGetArc();
  const contextMenu: NonNullable<Menu.ContextMenuProps["menu"]> = (props) => {
    const rows = getItem(props.keys);
    const arcKeys = rows.map(arc.task.arcKeyOf).filter((key) => key != null);
    if (arcKeys.length > 0 && arcKeys.length === rows.length)
      return <PlatformArc.ContextMenu {...props} keys={arcKeys} getItem={getArc} />;
    return (
      <ContextMenu
        keys={props.keys}
        getItem={getItem}
        subscribe={subscribe}
        onDelete={handleDelete}
        onStart={handleStart}
        onStop={handleStop}
        onEdit={handleEdit}
        onEnableDataSaving={handleEnableDataSaving}
        onDisableDataSaving={handleDisableDataSaving}
      />
    );
  };
  const handleListItemStopStart = (command: PlatformTask.Command, key: task.Key) =>
    handleCommand([key], command);
  return (
    <Menu.ContextMenu menu={contextMenu} {...menuProps}>
      <Toolbar.Content className={CSS.cls(CSS.B("task-toolbar"), menuProps.className)}>
        <Toolbar.Header>
          <Toolbar.Title>
            <Icon.Task />
            Tasks
          </Toolbar.Title>
          {hasCreatePermission && (
            <Toolbar.Actions>
              <Toolbar.CreateAction
                tooltip="Create task"
                onClick={() => openSelector()}
              />
            </Toolbar.Actions>
          )}
        </Toolbar.Header>
        <Toolbar.Body>
          <Select.Frame
            multiple
            data={data}
            getItem={getItem}
            subscribe={subscribe}
            value={selected}
            onChange={setSelected}
            onFetchMore={more}
            replaceOnSingle
          >
            <List.Scroll full="y" onContextMenu={menuProps.open}>
              <List.Items<task.Key, task.Task>
                emptyContent={answered && <EmptyContent />}
              >
                {({ key, ...p }) => (
                  <ListItem
                    key={key}
                    {...p}
                    onStopStart={handleListItemStopStart}
                    onSetDataSaving={setDataSaving}
                    onRename={rename}
                    onEdit={handleEdit}
                    onRenameArc={renameArc}
                    onEditArc={handleEditArc}
                  />
                )}
              </List.Items>
            </List.Scroll>
          </Select.Frame>
        </Toolbar.Body>
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
  useVisible: () => Access.useRetrieveGranted(task.TYPE_RESOURCE_ID),
};

// Reads Arcs from the client cache. Each Arc row loads its Arc.
const useGetArc = (): List.GetItem<arc.Key, arc.Arc> => {
  const client = Synnax.use();
  const getOne = (key: arc.Key): arc.Arc | undefined => {
    const cached = client?.arcs.watch({ key }).getCached();
    return query.isLive(cached) ? cached : undefined;
  };
  return List.createGetItem(getOne, (keys) =>
    keys.map(getOne).filter((a) => a != null),
  );
};

interface ListItemProps extends TaskListItemProps {
  onRenameArc: (params: { key: arc.Key; name: string }) => void;
  onEditArc: (key: arc.Key) => void;
}

const ListItem = ({
  onStopStart,
  onSetDataSaving,
  onRename,
  onEdit,
  onRenameArc,
  onEditArc,
  ...rest
}: ListItemProps) => {
  const tsk = List.useItem<task.Key, task.Task>(rest.itemKey);
  const arcKey = tsk == null ? undefined : arc.task.arcKeyOf(tsk);
  if (arcKey != null)
    return (
      <ArcItem {...rest} arcKey={arcKey} onRename={onRenameArc} onEdit={onEditArc} />
    );
  return (
    <TaskListItem
      {...rest}
      onStopStart={onStopStart}
      onSetDataSaving={onSetDataSaving}
      onRename={onRename}
      onEdit={onEdit}
    />
  );
};

interface TaskListItemProps extends List.ItemProps<task.Key> {
  onStopStart: (command: PlatformTask.Command, key: task.Key) => void;
  onSetDataSaving: (params: SetDataSavingParams) => void;
  onRename: (params: { key: task.Key; name: string }) => void;
  onEdit: (key: task.Key) => void;
}

const TaskListItem = ({
  onStopStart,
  onSetDataSaving,
  onRename,
  onEdit,
  ...rest
}: TaskListItemProps) => {
  const { itemKey } = rest;
  const { getIcon, parseType } = PlatformTask.useRegistry();
  const task_ = List.useItem<task.Key, task.Task>(itemKey);
  const hasUpdatePermission = Access.useUpdateGranted(task.resourceID(itemKey));
  const canControl = Framer.useCanCommand();
  const details = task_?.status?.details;
  let variant = task_?.status?.variant;
  const icon = getIcon(task_?.type ?? "");
  const isLoading = variant === "loading";
  const isRunning = details?.running === true;
  const isDrifted = task_ != null && task.drifted(task_.payload);
  const config = task_?.config;
  const dataSaving =
    config != null && typeof config === "object" && "dataSavingDisabled" in config
      ? config.dataSavingDisabled !== true
      : null;
  if (!isRunning && variant === "success") variant = "info";
  const handleStartStopClick = () => onStopStart(isRunning ? "stop" : "start", itemKey);
  const handleDataSavingClick = () =>
    onSetDataSaving({ key: itemKey, dataSaving: dataSaving !== true });
  const handleRename = (name: string) => onRename({ key: itemKey, name });
  const handleDoubleClick = () => onEdit(itemKey);
  return (
    <Select.Item
      {...rest}
      onDoubleClick={handleDoubleClick}
      justify="between"
      align="center"
    >
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
              onChange={hasUpdatePermission ? handleRename : undefined}
              allowDoubleClick={false}
              overflow="fade"
              weight={500}
            />
            {isDrifted && (
              <Tooltip.Dialog>
                <Text.Text level="small">Configuration changed since deploy</Text.Text>
                <Text.Text level="small" status="warning">
                  <Icon.Refresh />
                </Text.Text>
              </Tooltip.Dialog>
            )}
          </Flex.Box>
        </Flex.Box>
        <Text.Text level="tiny" color={8}>
          {parseType(task_?.type ?? "")}
        </Text.Text>
      </Flex.Box>
      {dataSaving != null && (
        <Button.Button
          variant="text"
          size="small"
          className={CSS.cls(
            CSS.BE("task", "data-saving"),
            isRunning &&
              dataSaving &&
              !isDrifted &&
              CSS.BEM("task", "data-saving", "active"),
          )}
          textColor={
            dataSaving ? "var(--pluto-primary-z)" : "var(--pluto-text-disabled)"
          }
          preventClick={!hasUpdatePermission}
          onClick={handleDataSavingClick}
          onDoubleClick={stopPropagation}
          tooltip={`Data saving ${dataSaving ? "enabled" : "disabled"}${
            hasUpdatePermission
              ? `. Click to ${dataSaving ? "disable" : "enable"}.`
              : ""
          }`}
        >
          {dataSaving ? <Icon.Database /> : <Icon.DatabaseOff />}
        </Button.Button>
      )}
      {canControl && (
        <Button.Button
          variant="outlined"
          size="small"
          status={isLoading ? "loading" : isRunning ? "error" : undefined}
          onClick={handleStartStopClick}
          onDoubleClick={stopPropagation}
          tooltip={`${isRunning ? "Stop" : "Start"} ${task_?.name ?? ""}`}
        >
          {isRunning ? <Icon.Stop /> : <Icon.Play />}
        </Button.Button>
      )}
    </Select.Item>
  );
};

interface ContextMenuProps extends Store.UseItemsProps<task.Key, task.Task> {
  onDelete: (keys: task.Key[]) => void;
  onStart: (keys: task.Key[]) => void;
  onStop: (keys: task.Key[]) => void;
  onEdit: (key: task.Key) => void;
  onEnableDataSaving: (keys: task.Key[]) => void;
  onDisableDataSaving: (keys: task.Key[]) => void;
}

const ContextMenu = ({
  keys,
  getItem,
  subscribe,
  onDelete,
  onStart,
  onStop,
  onEdit,
  onEnableDataSaving,
  onDisableDataSaving,
}: ContextMenuProps) => {
  const selectedTasks = Store.useItems({ keys, subscribe, getItem });
  const activeRange = Range.useResolve();
  const snapshotToActiveRange = useRangeSnapshot();
  const resourceIDs = task.resourceID(keys);
  // Deleting the row of an Arc deletes the Arc.
  const arcIDs = arc.resourceID(
    selectedTasks.map(arc.task.arcKeyOf).filter((key) => key != null),
  );
  const hasCreatePermission = Access.useCreateGranted(task.TYPE_RESOURCE_ID);
  const hasDeletePermission = Access.useDeleteGranted([...resourceIDs, ...arcIDs]);
  const hasUpdatePermission = Access.useUpdateGranted(resourceIDs);
  const canControl = Framer.useCanCommand();

  const canStart = selectedTasks.some(
    ({ status }) => status?.details.running === false,
  );
  const canStop = selectedTasks.some(({ status }) => status?.details.running === true);
  const redeployKeys = selectedTasks
    .filter((t) => t.status?.details.running === true && task.drifted(t.payload))
    .map(({ key }) => key);
  const someSelected = selectedTasks.length > 0;
  const isSingle = selectedTasks.length === 1;

  // Only tasks with a dataSavingDisabled field in their config (primarily read
  // tasks) are eligible for these menu items. Tasks without this field are excluded.
  const dataSavingTasks = selectedTasks.filter(
    ({ config }) =>
      config != null && typeof config === "object" && "dataSavingDisabled" in config,
  );
  const canEnableDataSaving = dataSavingTasks.some(
    ({ config }) =>
      config != null &&
      typeof config === "object" &&
      "dataSavingDisabled" in config &&
      config.dataSavingDisabled === true,
  );
  const canDisableDataSaving = dataSavingTasks.some(
    ({ config }) =>
      config != null &&
      typeof config === "object" &&
      "dataSavingDisabled" in config &&
      config.dataSavingDisabled === false,
  );

  const addStatus = Status.useAdder();
  const copyLinkToClipboard = Core.useCopyLinkToClipboard();

  const handleExport = Export.useResource();
  const handleLink = (key: task.Key) => {
    const name = selectedTasks.find((t) => t.key === key)?.name;
    if (name == null)
      return addStatus({
        variant: "error",
        message: "Failed to copy link",
        description: "The task no longer exists.",
      });
    copyLinkToClipboard({ name, resourceID: task.resourceID(key) });
  };
  // The menu of an Arc has no snapshot.
  const snapshotTasks = selectedTasks.filter(({ type }) => type !== arc.task.TYPE);
  const showSnapshotToActiveRange =
    activeRange?.variant === "persisted" && snapshotTasks.length > 0;
  return (
    <PlatformContextMenu.Menu>
      {canControl && (
        <>
          {canStart && (
            <Menu.Item itemKey="start" onClick={() => onStart(keys)}>
              <Icon.Play />
              Start
            </Menu.Item>
          )}
          {canStop && (
            <Menu.Item itemKey="stop" onClick={() => onStop(keys)}>
              <Icon.Stop />
              Stop
            </Menu.Item>
          )}
          {redeployKeys.length > 0 && (
            <Menu.Item
              className={CSS.BE("task", "redeploy-item")}
              itemKey="redeploy"
              onClick={() => onStart(redeployKeys)}
            >
              <Icon.Refresh />
              Redeploy
            </Menu.Item>
          )}
        </>
      )}
      <Menu.Divider />
      {hasUpdatePermission && isSingle && (
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
              tasks: snapshotTasks.map(({ name, resourceID: { key } }) => ({
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
            onClick={() => handleExport(task.resourceID(keys[0]))}
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
