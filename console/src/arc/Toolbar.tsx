// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/Toolbar.css";

import { type arc, UnexpectedError } from "@synnaxlabs/client";
import {
  Arc,
  Button,
  Flex,
  type Flux,
  Icon,
  List,
  Menu as PMenu,
  Select,
  Status,
  stopPropagation,
  Text,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { Editor } from "@/arc/editor";
import { remove } from "@/arc/slice";
import { translateGraphToConsole } from "@/arc/types/translate";
import { EmptyAction, Menu, Toolbar } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Ontology } from "@/ontology";

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  const handleClick = () => placeLayout(Editor.create());
  return (
    <EmptyAction
      message="No existing Arcs."
      action="Create an arc"
      onClick={handleClick}
    />
  );
};

const Content = () => {
  const [selected, setSelected] = useState<arc.Key[]>([]);
  const addStatus = Status.useAdder();
  const confirm = Modals.useConfirm();
  const menuProps = PMenu.useContextMenu();
  const placeLayout = Layout.usePlacer();
  const dispatch = useDispatch();
  const handleError = Status.useErrorHandler();

  const { data, getItem, subscribe, retrieve } = Arc.useList({});
  const { fetchMore } = List.usePager({ retrieve, pageSize: 1e3 });

  const { update: handleDelete } = Arc.useDelete({
    beforeUpdate: useCallback(
      async ({
        data: keys,
        rollbacks,
      }: Flux.BeforeUpdateParams<arc.Key | arc.Key[]>) => {
        setSelected([]);
        const keyArray = array.toArray(keys);
        if (keyArray.length === 0) return false;
        const confirmed = await confirm({
          message: `Are you sure you want to delete ${keyArray.length} automation(s)?`,
          description: "This action cannot be undone.",
          cancel: { label: "Cancel" },
          confirm: { label: "Delete", variant: "error" },
        });
        dispatch(Layout.remove({ keys: keyArray }));
        rollbacks.push(() => dispatch(Layout.remove({ keys: keyArray })));
        dispatch(remove({ keys: keyArray }));
        rollbacks.push(() => dispatch(remove({ keys: keyArray })));
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
      const graph = translateGraphToConsole(arc.graph);
      placeLayout(Editor.create({ key, name: arc.name, graph }));
    },
    [getItem, addStatus, placeLayout],
  );

  const { update: handleToggleDeploy } = Arc.useToggleDeploy();

  const rename = Modals.useRename();

  const handleCreate = useCallback(() => {
    handleError(async () => {
      const name = await rename({}, { icon: "Arc", name: "Arc.Create" });
      if (name == null) return;
      placeLayout(Editor.create({ name }));
    }, "Failed to create arc");
  }, [rename, handleError, placeLayout]);

  const { update: handleRename } = Arc.useRename({
    beforeUpdate: useCallback(
      async ({ data, rollbacks }: Flux.BeforeUpdateParams<Arc.RenameParams>) => {
        const { key, name } = data;
        const arc = getItem(key);
        if (arc == null) throw new UnexpectedError(`Arc with key ${key} not found`);
        const oldName = arc.name;
        if (arc.status?.details.running === true) {
          const confirmed = await confirm({
            message: `Are you sure you want to rename ${arc.name} to ${name}?`,
            description: `This will cause ${arc.name} to stop and be reconfigured.`,
            cancel: { label: "Cancel" },
            confirm: { label: "Rename", variant: "error" },
          });
          if (!confirmed) return false;
        }
        dispatch(Layout.rename({ key, name }));
        rollbacks.push(() => dispatch(Layout.rename({ key, name: oldName })));
        return data;
      },
      [dispatch, getItem],
    ),
  });

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
          <Toolbar.Title icon={<Icon.Arc />}>Arcs</Toolbar.Title>
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
                onRename={(name) => handleRename({ key, name })}
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
  tooltip: "Arcs",
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

  const variant = arc?.status?.variant;
  const isLoading = variant === "loading";
  const isRunning = arc?.status?.details.running === true;
  const isDeployed = arc?.deploy === true;

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
          {arc?.status?.message ?? (isDeployed ? "Started" : "Stopped")}
        </Text.Text>
      </Flex.Box>
      <Button.Button
        variant="outlined"
        status={isLoading ? "loading" : undefined}
        onClick={onToggleDeploy}
        onDoubleClick={stopPropagation}
        tooltip={`${isDeployed ? "Stop" : "Start"} ${arc?.name ?? ""}`}
      >
        {isRunning ? <Icon.Pause /> : <Icon.Play />}
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
  const canStop = arcs.some((arc) => arc.deploy === true);
  const someSelected = arcs.length > 0;
  const isSingle = arcs.length === 1;

  const handleChange = useMemo<PMenu.MenuProps["onChange"]>(
    () => ({
      start: () =>
        arcs.forEach((arc) => {
          if (!arc.deploy) onToggleDeploy(arc.key);
        }),
      stop: () =>
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
      {(canDeploy || canStop) && <PMenu.Divider />}
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
