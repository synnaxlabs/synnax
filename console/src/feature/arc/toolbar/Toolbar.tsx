// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/arc/toolbar/Toolbar.css";

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

import { Explorer } from "@/feature/arc/explorer";
import { Arc as PlatformArc } from "@/platform/arc";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Panel } from "@/platform/panel";
import { Toolbar } from "@/platform/toolbar";

interface EmptyContentProps {
  onCreate: () => void;
}

const EmptyContent = ({ onCreate }: EmptyContentProps) => {
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  return (
    <Empty.Action
      message="No existing Arcs."
      action={hasCreatePermission ? "Create an Arc" : undefined}
      onClick={onCreate}
    />
  );
};

const Content = () => {
  const [selected, setSelected] = useState<arc.Key[]>([]);
  const menuProps = Menu.useContextMenu();
  const openTab = Panel.useOpenTab();

  const { data, getItem, subscribe, retrieve } = Arc.useList({});
  const { fetchMore } = List.usePager({ retrieve, pageSize: 1e3 });

  const { update: handleRename } = PlatformArc.useRename(getItem);

  const handleEdit = useCallback(
    (key: arc.Key) => openTab({ variant: "resource", resource: arc.ontologyID(key) }),
    [openTab],
  );

  const create = PlatformArc.useCreate();

  const contextMenu = useCallback<NonNullable<Menu.ContextMenuProps["menu"]>>(
    (props) => <PlatformArc.ContextMenu {...props} getItem={getItem} />,
    [getItem],
  );

  return (
    <Menu.ContextMenu menu={contextMenu} {...menuProps}>
      <Toolbar.Content className={CSS(CSS.B("arc-toolbar"), menuProps.className)}>
        <Toolbar.Header padded>
          <Toolbar.Title icon={<Icon.Arc />}>Arcs</Toolbar.Title>
          <Actions handleCreate={create} />
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
            emptyContent={<EmptyContent onCreate={create} />}
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
  const openExplorer = Explorer.useOpenTab();
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
          onClick={openExplorer}
          variant="filled"
        >
          <Icon.Explore />
        </Toolbar.Action>
      )}
    </Toolbar.Actions>
  );
};

export const TOOLBAR: Nav.Toolbar = {
  key: "arc",
  icon: <Icon.Arc />,
  content: <Content />,
  trigger: ["A"],
  tooltip: "Arcs",
  sizeBounds: { lower: 225, upper: 400 },
  initialSize: 300,
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
  } = PlatformArc.useTask(itemKey, arcItem?.name ?? "");
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
