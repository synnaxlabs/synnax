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
  stopPropagation,
  Text,
  Tooltip,
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
      message="No Arcs"
      action={hasCreatePermission ? "Create Arc automation" : undefined}
      onClick={onCreate}
    />
  );
};

const Content = () => {
  const [selected, setSelected] = useState<arc.Key[]>([]);
  const menuProps = Menu.useContextMenu();
  const openTab = Panel.useOpenTab();

  const { data, getItem, subscribe, retrieve, answered } = Arc.useList({});
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
      <Toolbar.Content className={CSS.cls(CSS.B("arc-toolbar"), menuProps.className)}>
        <Toolbar.Header>
          <Toolbar.Title>
            <Icon.Arc />
            Arcs
          </Toolbar.Title>
          <Actions handleCreate={create} />
        </Toolbar.Header>
        <Toolbar.Body>
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
              emptyContent={answered && <EmptyContent onCreate={create} />}
              onContextMenu={menuProps.open}
            >
              {({ key, ...p }) => (
                <ArcListItem
                  key={key}
                  {...p}
                  onRename={handleRename}
                  onEdit={handleEdit}
                />
              )}
            </List.Items>
          </Select.Frame>
        </Toolbar.Body>
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
      {hasRetrievePermission && (
        <Toolbar.Action tooltip="Open Arc explorer" onClick={openExplorer}>
          <Icon.Explore />
        </Toolbar.Action>
      )}
      {hasCreatePermission && (
        <Toolbar.Action
          tooltip="Create Arc automation"
          onClick={handleCreate}
          variant="filled"
        >
          <Icon.Add />
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
  onRename: (params: { key: arc.Key; name: string }) => void;
  onEdit: (key: arc.Key) => void;
}

const ArcListItem = ({ onRename, onEdit, ...rest }: ArcListItemProps) => {
  const { itemKey } = rest;
  const arcItem = List.useItem<arc.Key, arc.Arc>(itemKey);
  const hasUpdatePermission = Access.useUpdateGranted(arc.ontologyID(itemKey));
  const handleRename = useCallback(
    (name: string) => onRename({ key: itemKey, name }),
    [onRename, itemKey],
  );
  const handleDoubleClick = useCallback(() => onEdit(itemKey), [onEdit, itemKey]);
  const {
    running,
    onStartStop,
    taskStatus: status,
  } = Arc.useTaskControls(itemKey, arcItem?.name ?? "");
  const drifted = Arc.useDrifted({ arcKey: itemKey });
  const isLoading = status.variant === "loading";
  let statusMessage = "Stopped";
  if (status.variant === "success" && running) statusMessage = "Running";
  else if (status.variant === "error") statusMessage = "Error";
  return (
    <Select.ListItem
      {...rest}
      onDoubleClick={handleDoubleClick}
      justify="between"
      align="center"
    >
      <Flex.Box y gap="small" grow className={CSS.BE("arc", "metadata")}>
        <Flex.Box x align="center" gap="small">
          <Status.Indicator
            variant={status.variant}
            className={CSS.BE("arc-list-item", "status")}
          />
          <Text.MaybeEditable
            id={`text-${itemKey}`}
            value={arcItem?.name ?? ""}
            onChange={hasUpdatePermission ? handleRename : undefined}
            allowDoubleClick={false}
            overflow="ellipsis"
            weight={500}
          />
          {drifted && (
            <Tooltip.Dialog>
              <Text.Text level="small">Configuration changed since deploy</Text.Text>
              <Text.Text level="small" status="warning">
                <Icon.Refresh />
              </Text.Text>
            </Tooltip.Dialog>
          )}
        </Flex.Box>
        <Text.Text level="small" status={status?.variant}>
          {statusMessage}
        </Text.Text>
      </Flex.Box>
      {hasUpdatePermission && (
        <Button.Button
          variant="outlined"
          size="small"
          status={isLoading ? "loading" : running ? "error" : undefined}
          onClick={onStartStop}
          onDoubleClick={stopPropagation}
          tooltip={`${running ? "Stop" : "Start"} ${arcItem?.name ?? ""}`}
        >
          {running ? <Icon.Stop /> : <Icon.Play />}
        </Button.Button>
      )}
    </Select.ListItem>
  );
};
