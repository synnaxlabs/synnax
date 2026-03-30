// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/Toolbar.css";

import { arc } from "@synnaxlabs/client";
import {
  Access,
  Arc,
  Button,
  Flex,
  Icon,
  List,
  Menu,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";

import { ContextMenu } from "@/arc/ContextMenu";
import { Editor } from "@/arc/editor";
import { EXPLORER_LAYOUT } from "@/arc/Explorer";
import { useRename, useTask } from "@/arc/hooks";
import { translateGraphToConsole } from "@/arc/types/translate";
import { EmptyAction, Toolbar } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";

interface EmptyContentProps {
  onCreate: () => void;
}

const EmptyContent = ({ onCreate }: EmptyContentProps) => {
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  return (
    <EmptyAction
      message="No existing Arcs."
      action={hasCreatePermission ? "Create an Arc" : undefined}
      onClick={onCreate}
    />
  );
};

const Content = () => {
  const [selected, setSelected] = useState<arc.Key[]>([]);
  const addStatus = Status.useAdder();
  const menuProps = Menu.useContextMenu();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();

  const { data, getItem, subscribe, retrieve } = Arc.useList({});
  const { fetchMore } = List.usePager({ retrieve, pageSize: 1e3 });

  const { update: handleRename } = useRename(getItem);

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

  const contextMenu = useCallback<NonNullable<Menu.ContextMenuProps["menu"]>>(
    (props) => <ContextMenu {...props} getItem={getItem} />,
    [getItem],
  );

  return (
    <Menu.ContextMenu menu={contextMenu} {...menuProps}>
      <Toolbar.Content className={CSS(CSS.B("arc-toolbar"), menuProps.className)}>
        <Toolbar.Header padded>
          <Toolbar.Title icon={<Icon.Arc />}>Arcs</Toolbar.Title>
          <Actions handleCreate={handleCreate} />
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
                onDoubleClick={() => handleEdit(key)}
              />
            )}
          </List.Items>
        </Select.Frame>
      </Toolbar.Content>
    </Menu.ContextMenu>
  );
};

interface ActionsProps {
  handleCreate: () => void;
}

const Actions = ({ handleCreate }: ActionsProps): ReactElement | null => {
  const placeLayout = Layout.usePlacer();
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  const hasRetrievePermission = Access.useRetrieveGranted(arc.TYPE_ONTOLOGY_ID);
  if (!hasCreatePermission && !hasRetrievePermission) return null;
  return (
    <Toolbar.Actions>
      {hasCreatePermission && (
        <Toolbar.Action tooltip="Create Arc" onClick={handleCreate}>
          <Icon.Add />
        </Toolbar.Action>
      )}
      {hasRetrievePermission && (
        <Toolbar.Action
          tooltip="Open Arc Explorer"
          onClick={() => placeLayout(EXPLORER_LAYOUT)}
          variant="filled"
        >
          <Icon.Explore />
        </Toolbar.Action>
      )}
    </Toolbar.Actions>
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
  const hasUpdatePermission = Access.useUpdateGranted(arc.ontologyID(itemKey));
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
            onChange={hasUpdatePermission ? onRename : undefined}
            allowDoubleClick={false}
            overflow="ellipsis"
            weight={500}
          />
        </Flex.Box>
        <Text.Text level="small" status={status?.variant}>
          {statusMessage}
        </Text.Text>
      </Flex.Box>
      {hasUpdatePermission && (
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
