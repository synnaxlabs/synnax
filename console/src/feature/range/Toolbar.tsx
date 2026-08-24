// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/Toolbar.css";

import { ranger } from "@synnaxlabs/client";
import {
  Access,
  Component,
  Flex,
  Haul,
  Icon,
  List as BaseList,
  Menu,
  Ranger,
  Select,
  Tag,
  Telem,
  Text,
  Tooltip,
} from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { ContextMenu } from "@/feature/range/ContextMenu";
import { Explorer } from "@/feature/range/explorer";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Range } from "@/platform/range";
import { Toolbar } from "@/platform/toolbar";
import { Session } from "@/session";

const NoRanges = (): ReactElement => {
  const openExplorer = Explorer.useOpenTab();
  const hasRetrievePermission = Access.useRetrieveGranted(ranger.TYPE_ONTOLOGY_ID);
  return (
    <Empty.Action
      message="No favorited ranges"
      action={hasRetrievePermission ? "Open range explorer" : undefined}
      onClick={openExplorer}
    />
  );
};

const List = (): ReactElement => {
  const dispatch = Session.useDispatch();
  const activeRange = Session.Range.useSelectState();
  const data = Session.Range.useSelectStaticKeys();

  const handleSelect = (key: string): void => {
    dispatch(Session.Range.select(key));
  };

  const dropProps = Haul.useDrop({
    type: "range_toolbar",
    canDrop: Ranger.canDropHaulItem,
    onDrop: ({ items }) => {
      const dropped = Ranger.filterHaulItems(items);
      const ranges = dropped.map<Session.Range.StaticState>(({ data }) => ({
        ...data,
        persisted: true,
        variant: "static",
      }));
      Session.Range.add(ranges);
      return dropped;
    },
  });

  const menuProps = Menu.useContextMenu();

  return (
    <Select.Frame<string, Session.Range.StaticState>
      data={data}
      value={activeRange?.key}
      onChange={handleSelect}
    >
      <Menu.ContextMenu menu={(p) => <ContextMenu {...p} />} {...menuProps} />
      <BaseList.Items
        full="y"
        emptyContent={<NoRanges />}
        {...dropProps}
        onContextMenu={menuProps.open}
      >
        {listItem}
      </BaseList.Items>
    </Select.Frame>
  );
};

const listItem = Component.renderProp((props: BaseList.ItemProps<string>) => {
  const { itemKey } = props;
  const entry = Range.useResolve(itemKey);
  const isLocal = entry != null && entry.variant !== "persisted";
  const labels = Ranger.useLabels(isLocal ? null : itemKey) ?? [];
  const onRename = Session.Range.useRename();
  const hasUpdatePermission = Access.useUpdateGranted(ranger.ontologyID(itemKey));
  if (entry == null || entry.variant === "dynamic") return null;
  const { key, name, timeRange } = entry;
  return (
    <Select.ListItem className={CSS.B("range-list-item")} {...props} gap="small" y>
      {isLocal && (
        <Tooltip.Dialog location="left">
          <Text.Text level="small">This range is local.</Text.Text>
          <Text.Text className="save-button" weight={700} level="small" color={11}>
            L
          </Text.Text>
        </Tooltip.Dialog>
      )}
      <Flex.Box
        x
        align="center"
        gap="small"
        className={CSS.BE("range-list-item", "name")}
      >
        <Ranger.StageIcon timeRange={timeRange} />
        <Text.MaybeEditable
          id={`text-${key}`}
          level="p"
          value={name}
          overflow="fade"
          onChange={
            hasUpdatePermission ? (name) => onRename.update({ key, name }) : undefined
          }
          allowDoubleClick={false}
        />
      </Flex.Box>
      <Telem.Text.TimeRange level="small">{timeRange}</Telem.Text.TimeRange>
      {labels.length > 0 && (
        <Flex.Box x gap="small" wrap className={CSS.B("range-list-item-labels")}>
          {labels.map((l) => (
            <Tag.Tag key={l.key} size="tiny" color={l.color}>
              {l.name}
            </Tag.Tag>
          ))}
        </Flex.Box>
      )}
    </Select.ListItem>
  );
});

const Actions = (): ReactElement | null => {
  const openExplorer = Explorer.useOpenTab();
  const openCreate = Range.useCreateModal();
  const hasCreatePermission = Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID);
  const hasRetrievePermission = Access.useRetrieveGranted(ranger.TYPE_ONTOLOGY_ID);
  if (!hasCreatePermission && !hasRetrievePermission) return null;
  return (
    <Toolbar.Actions>
      {hasRetrievePermission && (
        <Toolbar.Action tooltip="Open range explorer" onClick={openExplorer}>
          <Icon.Explore />
        </Toolbar.Action>
      )}
      {hasCreatePermission && (
        <Toolbar.Action
          tooltip="Create range"
          onClick={() => openCreate()}
          variant="filled"
        >
          <Icon.Add />
        </Toolbar.Action>
      )}
    </Toolbar.Actions>
  );
};

const Content = (): ReactElement => (
  <Toolbar.Content>
    <Toolbar.Header>
      <Toolbar.Title>
        <Icon.Range />
        Ranges
      </Toolbar.Title>
      <Actions />
    </Toolbar.Header>
    <Toolbar.Body>
      <List />
    </Toolbar.Body>
  </Toolbar.Content>
);

export const TOOLBAR: Nav.Toolbar = {
  key: "range",
  icon: <Icon.Range />,
  content: <Content />,
  tooltip: "Ranges",
  trigger: ["R"],
  initialSize: 300,
  sizeBounds: { lower: 175, upper: 400 },
  useVisible: () => Access.useRetrieveGranted(ranger.TYPE_ONTOLOGY_ID),
};
