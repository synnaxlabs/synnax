// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/Toolbar.css";

import { arc, UnexpectedError } from "@synnaxlabs/client";
import {
  Access,
  Arc,
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
import { array } from "@synnaxlabs/x";
import { useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { Editor } from "@/arc/editor";
import { useTask } from "@/arc/hooks";
import { remove } from "@/arc/slice";
import { translateGraphToConsole } from "@/arc/types/translate";
import { EmptyAction, Menu, Toolbar } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";

interface EmptyContentProps {
  onCreate: () => void;
}

const EmptyContent = ({ onCreate }: EmptyContentProps) => {
  const canCreateArc = Access.useUpdateGranted(arc.TYPE_ONTOLOGY_ID);
  return (
    <EmptyAction
      message="No existing Arcs."
      action={canCreateArc ? "Create an Arc" : undefined}
      onClick={onCreate}
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
  const canCreateArc = Access.useUpdateGranted(arc.TYPE_ONTOLOGY_ID);

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

  const { update: handleRename } = Arc.useRename({
    beforeUpdate: useCallback(
      async ({
        data,
        rollbacks,
        store,
        client,
      }: Flux.BeforeUpdateParams<Arc.RenameParams, false, Arc.FluxSubStore>) => {
        const { key, name } = data;
        const tsk = await Arc.retrieveTask({ store, client, query: { arcKey: key } });
        const arc = getItem(key);
        if (arc == null) throw new UnexpectedError(`Arc with key ${key} not found`);
        const oldName = arc.name;
        if (tsk?.status?.details.running === true) {
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

  const handleEdit = useCallback(
    (key: arc.Key) => {
      const retrieved = getItem(key);
      if (retrieved == null)
        return addStatus({
          variant: "error",
          message: "Failed to open Arc editor",
          description: `Arc with key ${key} not found`,
        });
      const { name, text, mode } = retrieved;
      const graph = translateGraphToConsole(retrieved.graph);
      placeLayout(Editor.create({ key, name, graph, text, mode }));
    },
    [getItem, addStatus, placeLayout],
  );

  const createArc = Editor.useCreateModal();

  const handleCreate = useCallback(() => {
    handleError(async () => {
      const result = await createArc({});
      if (result == null) return;
      placeLayout(Editor.create({ name: result.name, mode: result.mode }));
    }, "Failed to create Arc program");
  }, [createArc, handleError, placeLayout]);

  const contextMenu = useCallback<NonNullable<PMenu.ContextMenuProps["menu"]>>(
    ({ keys }) => (
      <ContextMenu
        keys={keys}
        arcs={getItem(keys)}
        onDelete={handleDelete}
        onEdit={handleEdit}
      />
    ),
    [handleDelete, handleEdit, getItem],
  );

  return (
    <PMenu.ContextMenu menu={contextMenu} {...menuProps}>
      <Toolbar.Content className={CSS(CSS.B("arc-toolbar"), menuProps.className)}>
        <Toolbar.Header padded>
          <Toolbar.Title icon={<Icon.Arc />}>Arcs</Toolbar.Title>
          {canCreateArc && (
            <Toolbar.Actions>
              <Toolbar.Action onClick={handleCreate}>
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
          <List.Items<arc.Key, arc.Arc>
            full="y"
            emptyContent={<EmptyContent onCreate={handleCreate} />}
            onContextMenu={menuProps.open}
          >
            {({ key, ...p }) => (
              <ArcListItem
                key={key}
                {...p}
                onRename={(name) => handleRename({ key, name })}
                onEdit={() => handleEdit(key)}
              />
            )}
          </List.Items>
        </Select.Frame>
      </Toolbar.Content>
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
  useVisible: () => Access.useRetrieveGranted(arc.TYPE_ONTOLOGY_ID),
};

interface ArcListItemProps extends List.ItemProps<arc.Key> {
  onRename: (name: string) => void;
  onEdit: () => void;
}

const ArcListItem = ({ onRename, onEdit, ...rest }: ArcListItemProps) => {
  const { itemKey } = rest;
  const arcItem = List.useItem<arc.Key, arc.Arc>(itemKey);
  const hasEditPermission = Access.useUpdateGranted(arc.ontologyID(itemKey));
  const {
    running,
    onStartStop,
    taskStatus: status,
  } = useTask(itemKey, arcItem?.name ?? "");
  let statusMessage = "Stopped";
  if (status.variant === "success" && running) statusMessage = "Running";
  else if (status.variant === "error") statusMessage = "Error";
  return (
    <Select.ListItem {...rest} justify="between" align="center">
      <Flex.Box y gap="small" grow className={CSS.BE("arc", "metadata")}>
        <Flex.Box x align="center" gap="small">
          <Status.Indicator
            variant={status.variant}
            style={{ fontSize: "2rem", minWidth: "2rem" }}
          />
          <Text.MaybeEditable
            id={`text-${itemKey}`}
            value={arcItem?.name ?? ""}
            onChange={hasEditPermission ? onRename : undefined}
            allowDoubleClick={false}
            overflow="ellipsis"
            weight={500}
          />
        </Flex.Box>
        <Text.Text level="small" status={status?.variant}>
          {statusMessage}
        </Text.Text>
      </Flex.Box>
      {hasEditPermission && (
        <Button.Button
          variant="outlined"
          onClick={onStartStop}
          tooltip={`${running ? "Stop" : "Start"} ${arcItem?.name ?? ""}`}
        >
          {running ? <Icon.Pause /> : <Icon.Play />}
        </Button.Button>
      )}
    </Select.ListItem>
  );
};

interface ContextMenuProps {
  keys: arc.Key[];
  arcs: arc.Arc[];
  onDelete: (keys: arc.Key | arc.Key[]) => void;
  onEdit: (key: arc.Key) => void;
}

const ContextMenu = ({ keys, arcs, onDelete, onEdit }: ContextMenuProps) => {
  const ids = arc.ontologyID(keys);
  const canDeleteAccess = Access.useDeleteGranted(ids);
  const canEditAccess = Access.useUpdateGranted(ids);
  const someSelected = arcs.length > 0;
  const isSingle = arcs.length === 1;

  const handleChange = useMemo<PMenu.MenuProps["onChange"]>(
    () => ({
      edit: () => isSingle && onEdit(arcs[0].key),
      rename: () => isSingle && Text.edit(`text-${arcs[0].key}`),
      delete: () => onDelete(keys),
    }),
    [arcs, onEdit, onDelete, isSingle, keys],
  );

  return (
    <PMenu.Menu level="small" gap="small" onChange={handleChange}>
      {canEditAccess && isSingle && (
        <>
          <PMenu.Item itemKey="edit">
            <Icon.Edit />
            Edit Arc
          </PMenu.Item>
          <PMenu.Divider />
          <Menu.RenameItem />
          <PMenu.Divider />
        </>
      )}
      {canDeleteAccess && someSelected && (
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
