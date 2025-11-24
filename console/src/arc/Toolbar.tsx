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
import { useCallback, useState } from "react";
import { useDispatch } from "react-redux";

import { Editor } from "@/arc/editor";
import { EXPLORER_LAYOUT } from "@/arc/Explorer";
import { contextMenuRenderProp } from "@/arc/list/ContextMenu";
import { translateGraphToConsole } from "@/arc/types/translate";
import { EmptyAction, Toolbar } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";

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
        console.log("handleRename", data);
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

  return (
    <PMenu.ContextMenu menu={contextMenuRenderProp} {...menuProps}>
      <Toolbar.Content className={CSS(CSS.B("arc-toolbar"), menuProps.className)}>
        <Toolbar.Header padded>
          <Toolbar.Title icon={<Icon.Arc />}>Arcs</Toolbar.Title>
          <Toolbar.Actions>
            <Toolbar.Action onClick={handleCreate} tooltip="Create Arc">
              <Icon.Add />
            </Toolbar.Action>
            <Toolbar.Action
              onClick={() => placeLayout(EXPLORER_LAYOUT)}
              tooltip="Open Arc Explorer"
              variant="filled"
            >
              <Icon.Explore />
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
