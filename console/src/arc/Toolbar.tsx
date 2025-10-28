// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/Toolbar.css";

import { type arc } from "@synnaxlabs/client";
import {
  Arc,
  Button,
  Flex,
  Icon,
  List,
  Menu as PMenu,
  Select,
  Status,
  stopPropagation,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { useCallback, useMemo, useState } from "react";

import { createEditor } from "@/arc/editor";
import { EmptyAction, Menu, Toolbar } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Ontology } from "@/ontology";

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  const handleClick = () => placeLayout(createEditor());
  return (
    <EmptyAction
      message="No existing Arc automations."
      action="Create an automation"
      onClick={handleClick}
    />
  );
};

const Content = () => {
  const client = Synnax.use();
  const [selected, setSelected] = useState<arc.Key[]>([]);
  const addStatus = Status.useAdder();
  const confirm = Modals.useConfirm();
  const menuProps = PMenu.useContextMenu();
  const placeLayout = Layout.usePlacer();

  const { data, getItem, subscribe, retrieve } = Arc.useList({});
  const { fetchMore } = List.usePager({ retrieve, pageSize: 1e3 });

  const { update: handleDelete } = Arc.useDelete({
    beforeUpdate: useCallback(
      async ({ data: keys }: { data: arc.Key | arc.Key[] }) => {
        setSelected([]);
        const keyArray = array.toArray(keys);
        if (keyArray.length === 0) return false;
        const confirmed = await confirm({
          message: `Are you sure you want to delete ${keyArray.length} automation(s)?`,
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
    (key: arc.Key) => {
      const arc = getItem(key);
      if (arc == null)
        return addStatus({
          variant: "error",
          message: "Failed to open Arc editor",
          description: `Arc with key ${key} not found`,
        });
      placeLayout(createEditor({ key, name: arc.name }));
    },
    [getItem, addStatus, placeLayout],
  );

  const { update: handleToggleDeploy } = Arc.useToggleDeploy();

  const { update: handleRename } = Arc.useRename({
    beforeUpdate: useCallback(
      async ({ data: { key, name } }: { data: { key: arc.Key; name: string } }) => {
        const arc = getItem(key);
        if (arc == null) return false;
        return { key, name };
      },
      [getItem],
    ),
  });

  const handleRename = useCallback(
    async (key: arc.Key, name: string) => {
      const arc = getItem(key);
      if (arc == null || client == null) return;

      const isRunning =
        arc.status &&
        typeof arc.status === "object" &&
        "details" in arc.status &&
        arc.status.details &&
        typeof arc.status.details === "object" &&
        "running" in arc.status.details &&
        arc.status.details.running === true;

      if (isRunning) {
        const confirmed = await confirm({
          message: `Are you sure you want to rename ${arc.name} to ${name}?`,
          description: `This will cause ${arc.name} to stop and be reconfigured.`,
          cancel: { label: "Cancel" },
          confirm: { label: "Rename", variant: "error" },
        });
        if (!confirmed) return;
      }

      try {
        await client.arcs.create({
          ...arc,
          name,
        });
      } catch (error) {
        addStatus({
          variant: "error",
          message: "Failed to rename automation",
          description: error instanceof Error ? error.message : "Unknown error",
        });
      }
    },
    [client, getItem, addStatus, confirm],
  );

  const contextMenu = useCallback<NonNullable<PMenu.ContextMenuProps["menu"]>>(
    ({ keys }) => (
      <ContextMenu
        keys={keys}
        arcs={getItem(keys)}
        onDelete={handleDelete}
        onEdit={handleEdit}
        onToggleDeploy={(key) => handleToggleDeploy(key)}
      />
    ),
    [handleDelete, handleEdit, handleToggleDeploy, handleRename, getItem],
  );

  return (
    <PMenu.ContextMenu menu={contextMenu} {...menuProps}>
      <Ontology.Toolbar className={CSS(CSS.B("arc-toolbar"), menuProps.className)}>
        <Toolbar.Header padded>
          <Toolbar.Title icon={<Icon.Arc />}>Arc Automations</Toolbar.Title>
          <Toolbar.Actions>
            <Toolbar.Action onClick={() => placeLayout(createEditor())}>
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
          <List.Items<arc.Key, arc.Arc>
            full="y"
            emptyContent={<EmptyContent />}
            onContextMenu={menuProps.open}
          >
            {({ key, ...p }) => (
              <ArcListItem
                key={key}
                {...p}
                onToggleDeploy={() => handleToggleDeploy(key)}
                onRename={(name) => handleRename(key, name)}
                onDoubleClick={() => handleEdit(key)}
              />
            )}
          </List.Items>
        </Select.Frame>
      </Ontology.Toolbar>
    </PMenu.ContextMenu>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "arc",
  icon: <Icon.Arc />,
  content: <Content />,
  trigger: ["A"],
  tooltip: "Arc Automations",
  initialSize: 300,
  minSize: 225,
  maxSize: 400,
};

interface ArcListItemProps extends List.ItemProps<arc.Key> {
  onToggleDeploy: () => void;
  onRename: (name: string) => void;
}

const ArcListItem = ({ onToggleDeploy, onRename, ...rest }: ArcListItemProps) => {
  const { itemKey } = rest;
  const arc = List.useItem<arc.Key, arc.Arc>(itemKey);

  let variant = arc?.status?.variant;
  const isLoading = variant === "loading";
  const isRunning =
    arc?.status &&
    typeof arc.status === "object" &&
    "details" in arc.status &&
    arc.status.details &&
    typeof arc.status.details === "object" &&
    "running" in arc.status.details &&
    arc.status.details.running === true;
  const isDeploy = arc?.deploy === true;

  // Use actual status variant and message, not inferred from deploy flag
  if (!isRunning && variant === "success") variant = "info";

  const handleDeployClick = useCallback<NonNullable<Button.ButtonProps["onClick"]>>(
    (e) => {
      e.stopPropagation();
      onToggleDeploy();
    },
    [onToggleDeploy],
  );

  return (
    <Select.ListItem {...rest} justify="between" align="center">
      <Flex.Box y gap="small" grow className={CSS.BE("arc", "metadata")}>
        <Flex.Box x align="center" gap="small">
          <Status.Indicator
            variant={variant}
            style={{ fontSize: "2rem", minWidth: "2rem" }}
          />
          <Text.MaybeEditable
            id={`text-${itemKey}`}
            value={arc?.name ?? ""}
            onChange={onRename}
            allowDoubleClick={false}
            overflow="ellipsis"
            weight={500}
          />
        </Flex.Box>
        <Text.Text level="small" color={10}>
          {arc?.status?.message ?? (isDeploy ? "Deployed" : "Not deployed")}
        </Text.Text>
      </Flex.Box>
      <Button.Button
        variant="outlined"
        status={isLoading ? "loading" : undefined}
        onClick={handleDeployClick}
        onDoubleClick={stopPropagation}
        tooltip={`${isDeploy ? "Undeploy" : "Deploy"} ${arc?.name ?? ""}`}
      >
        {isDeploy ? <Icon.Pause /> : <Icon.Play />}
      </Button.Button>
    </Select.ListItem>
  );
};

interface ContextMenuProps {
  keys: arc.Key[];
  arcs: arc.Arc[];
  onDelete: (keys: arc.Key | arc.Key[]) => void;
  onEdit: (key: arc.Key) => void;
  onToggleDeploy: (key: arc.Key) => void;
}

const ContextMenu = ({
  keys,
  arcs,
  onDelete,
  onEdit,
  onToggleDeploy,
}: ContextMenuProps) => {
  const canDeploy = arcs.some((arc) => arc.deploy === false);
  const canUndeploy = arcs.some((arc) => arc.deploy === true);
  const someSelected = arcs.length > 0;
  const isSingle = arcs.length === 1;

  const handleChange = useMemo<PMenu.MenuProps["onChange"]>(
    () => ({
      deploy: () =>
        arcs.forEach((arc) => {
          if (!arc.deploy) onToggleDeploy(arc.key);
        }),
      undeploy: () =>
        arcs.forEach((arc) => {
          if (arc.deploy) onToggleDeploy(arc.key);
        }),
      edit: () => isSingle && onEdit(arcs[0].key),
      rename: () => isSingle && Text.edit(`text-${arcs[0].key}`),
      delete: () => onDelete(keys),
    }),
    [arcs, onToggleDeploy, onEdit, onDelete, isSingle, keys],
  );

  return (
    <PMenu.Menu level="small" gap="small" onChange={handleChange}>
      {canDeploy && (
        <PMenu.Item itemKey="deploy">
          <Icon.Play />
          Deploy
        </PMenu.Item>
      )}
      {canUndeploy && (
        <PMenu.Item itemKey="undeploy">
          <Icon.Pause />
          Undeploy
        </PMenu.Item>
      )}
      {(canDeploy || canUndeploy) && <PMenu.Divider />}
      {isSingle && (
        <>
          <PMenu.Item itemKey="edit">
            <Icon.Edit />
            Edit automation
          </PMenu.Item>
          <PMenu.Divider />
          <Menu.RenameItem />
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
