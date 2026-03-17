// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type agent } from "@synnaxlabs/client";
import {
  Agent as AgentFlux,
  Button,
  Flex,
  type Flux,
  Icon,
  List,
  Menu as PMenu,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { array, type status } from "@synnaxlabs/x";
import { useCallback, useMemo, useState } from "react";

import { useCreateModal } from "@/agent/CreateModal";
import { create as createEditor } from "@/agent/editor/Editor";
import { useTask } from "@/arc/hooks";
import { EmptyAction, Menu, Toolbar } from "@/components";
import { CSS } from "@/css";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { usePlacer } from "@/layout/usePlacer";

const STATUS_VARIANT: Record<string, status.Variant> = {
  running: "success",
  error: "error",
  generating: "loading",
  stopped: "disabled",
};

interface AgentListItemProps extends List.ItemProps<agent.Key> {
  onEdit: () => void;
}

const AgentListItem = ({ onEdit, ...rest }: AgentListItemProps) => {
  const a = List.useItem<agent.Key, agent.Agent>(rest.itemKey);
  const arcKey = a?.arcKey ?? "";
  const zeroUUID = "00000000-0000-0000-0000-000000000000";
  const hasArc = arcKey !== "" && arcKey !== zeroUUID;
  const { running, onStartStop, taskStatus } = useTask(arcKey, a?.name ?? "Agent");

  let statusMessage = "Stopped";
  if (hasArc && taskStatus.variant === "success" && running) statusMessage = "Running";
  else if (hasArc && taskStatus.variant === "error") statusMessage = "Error";
  else if (a?.state === "generating") statusMessage = "Generating";

  const variant = hasArc
    ? taskStatus.variant
    : (STATUS_VARIANT[a?.state ?? "stopped"] ?? "disabled");

  return (
    <Select.ListItem {...rest} justify="between" align="center">
      <Flex.Box y gap="small" grow>
        <Flex.Box x align="center" gap="small">
          <Status.Indicator
            variant={variant}
            style={{ fontSize: "2rem", minWidth: "2rem" }}
          />
          <Text.Text level="p" weight={500} overflow="ellipsis">
            {a?.name ?? "Agent"}
          </Text.Text>
        </Flex.Box>
        <Text.Text level="small" status={variant}>
          {statusMessage}
        </Text.Text>
      </Flex.Box>
      {hasArc && (
        <Button.Button
          variant="outlined"
          onClick={onStartStop}
          tooltip={`${running ? "Stop" : "Start"} ${a?.name ?? "Agent"}`}
        >
          {running ? <Icon.Pause /> : <Icon.Play />}
        </Button.Button>
      )}
    </Select.ListItem>
  );
};

interface EmptyContentProps {
  onCreate: () => void;
}

const EmptyContent = ({ onCreate }: EmptyContentProps) => (
  <EmptyAction
    message="No existing Agents."
    action="Create an Agent"
    onClick={onCreate}
  />
);

const Content = () => {
  const [selected, setSelected] = useState<agent.Key[]>([]);
  const placeLayout = usePlacer();
  const createModal = useCreateModal();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const confirm = Modals.useConfirm();
  const menuProps = PMenu.useContextMenu();

  const { data, getItem, subscribe, retrieve } = AgentFlux.useList({});
  const { fetchMore } = List.usePager({ retrieve, pageSize: 1e3 });

  const { update: handleDelete } = AgentFlux.useDelete({
    beforeUpdate: useCallback(
      async ({
        data: keys,
        rollbacks,
      }: Flux.BeforeUpdateParams<agent.Key | agent.Key[]>) => {
        setSelected([]);
        const keyArray = array.toArray(keys);
        if (keyArray.length === 0) return false;
        const confirmed = await confirm({
          message: `Are you sure you want to delete ${keyArray.length} agent(s)?`,
          description: "This action cannot be undone.",
          cancel: { label: "Cancel" },
          confirm: { label: "Delete", variant: "error" },
        });
        if (!confirmed) return false;
        return keys;
      },
      [confirm],
    ),
    afterFailure: ({ status }) => addStatus(status),
  });

  const handleEdit = useCallback(
    (key: agent.Key) => {
      const a = getItem(key);
      placeLayout(createEditor({ key, name: a?.name ?? "Agent" }));
    },
    [getItem, placeLayout],
  );

  const handleCreate = useCallback(() => {
    handleError(async () => {
      const result = await createModal({});
      if (result == null) return;
      placeLayout(createEditor({ key: result.agent.key, name: result.agent.name }));
    }, "Failed to create Agent");
  }, [createModal, handleError, placeLayout]);

  const contextMenu = useCallback<NonNullable<PMenu.ContextMenuProps["menu"]>>(
    ({ keys }) => (
      <ContextMenu keys={keys} onDelete={handleDelete} onEdit={handleEdit} />
    ),
    [handleDelete, handleEdit],
  );

  return (
    <PMenu.ContextMenu menu={contextMenu} {...menuProps}>
      <Toolbar.Content className={menuProps.className}>
        <Toolbar.Header padded>
          <Toolbar.Title icon={<Icon.Auto />}>Agents</Toolbar.Title>
          <Toolbar.Actions>
            <Toolbar.Action onClick={handleCreate}>
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
          <List.Items<agent.Key, agent.Agent>
            full="y"
            emptyContent={<EmptyContent onCreate={handleCreate} />}
            onContextMenu={menuProps.open}
          >
            {({ key, ...p }) => (
              <AgentListItem
                key={key}
                {...p}
                onEdit={() => handleEdit(key)}
                onDoubleClick={() => handleEdit(key)}
              />
            )}
          </List.Items>
        </Select.Frame>
      </Toolbar.Content>
    </PMenu.ContextMenu>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "agent",
  icon: <Icon.Auto />,
  content: <Content />,
  trigger: ["G"],
  tooltip: "Agents",
  initialSize: 300,
  minSize: 225,
  maxSize: 400,
};

interface ContextMenuProps {
  keys: agent.Key[];
  onDelete: (keys: agent.Key | agent.Key[]) => void;
  onEdit: (key: agent.Key) => void;
}

const ContextMenu = ({ keys, onDelete, onEdit }: ContextMenuProps) => {
  const isSingle = keys.length === 1;
  const someSelected = keys.length > 0;

  const handleChange = useMemo<PMenu.MenuProps["onChange"]>(
    () => ({
      edit: () => isSingle && onEdit(keys[0]),
      delete: () => onDelete(keys),
    }),
    [keys, onEdit, onDelete, isSingle],
  );

  return (
    <PMenu.Menu level="small" gap="small" onChange={handleChange}>
      {isSingle && (
        <>
          <PMenu.Item itemKey="edit">
            <Icon.Edit />
            Edit Agent
          </PMenu.Item>
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
