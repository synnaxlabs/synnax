// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Toolbar.css";

import { DisconnectedError, ranger } from "@synnaxlabs/client";
import {
  Component,
  Flex,
  Haul,
  Icon,
  List as CoreList,
  Menu as PMenu,
  Ranger,
  Select,
  Status,
  Synnax,
  Tag,
  Text,
  Tooltip,
} from "@synnaxlabs/pluto";
import { type ReactElement } from "react";
import { useDispatch, useStore } from "react-redux";

import { Toolbar } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { ContextMenu } from "@/range/ContextMenu";
import { CREATE_LAYOUT } from "@/range/Create";
import { select, useSelect, useSelectKeys } from "@/range/selectors";
import { add, rename, setActive, type StaticRange } from "@/range/slice";
import { type RootState } from "@/store";

const NoRanges = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  const handleLinkClick = () => {
    placeLayout(CREATE_LAYOUT);
  };
  return (
    <Flex.Box
      empty
      style={{ height: "100%", position: "relative", padding: "1rem" }}
      className={CSS.B("range-toolbar-no-ranges")}
    >
      <Flex.Box y style={{ height: "100%" }} gap="medium">
        <Text.Text>No ranges loaded.</Text.Text>
        <Text.Text onClick={handleLinkClick}>Create a Range</Text.Text>
      </Flex.Box>
    </Flex.Box>
  );
};

const List = (): ReactElement => {
  const dispatch = useDispatch();
  const activeRange = useSelect();
  const data = useSelectKeys();

  const handleSelect = (key: string): void => {
    dispatch(setActive(key));
  };

  const dropProps = Haul.useDrop({
    type: "range-toolbar",
    canDrop: Haul.canDropOfType(ranger.ONTOLOGY_TYPE),
    onDrop: ({ items }) => {
      const ranges = items.map(
        ({ data, key }) =>
          ({
            key,
            name: data?.name,
            variant: "static",
            persisted: true,
            timeRange: data?.timeRange,
          }) as StaticRange,
      );
      dispatch(add({ ranges }));
      return items;
    },
  });

  const menuProps = PMenu.useContextMenu();

  return (
    <Select.Frame<string, StaticRange>
      data={data}
      value={activeRange?.key}
      onChange={handleSelect}
    >
      <PMenu.ContextMenu menu={(p) => <ContextMenu {...p} />} {...menuProps} />
      <CoreList.Items
        emptyContent={<NoRanges />}
        {...dropProps}
        onContextMenu={menuProps.open}
      >
        {listItem}
      </CoreList.Items>
    </Select.Frame>
  );
};

export const useRename = (key: string) => {
  const dispatch = useDispatch();
  const store = useStore<RootState>();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  return (name: string) => {
    const rng = select(store.getState(), key);
    dispatch(rename({ key, name }));
    if (rng != null && !rng.persisted) return;
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      await client.ranges.rename(key, name);
    }, `Failed to rename range to ${name}`);
  };
};

const listItem = Component.renderProp((props: CoreList.ItemProps<string>) => {
  const { itemKey } = props;
  const entry = useSelect(itemKey);
  const labels = Ranger.useLabels(itemKey)?.data ?? [];
  const onRename = useRename(itemKey);
  if (entry == null || entry.variant === "dynamic") return null;
  const { key, name, timeRange, persisted } = entry;

  return (
    <Select.ListItem className={CSS.B("range-list-item")} {...props} gap="small" y>
      {!persisted && (
        <Tooltip.Dialog location="left">
          <Text.Text level="small">This range is local.</Text.Text>
          <Text.Text className="save-button" weight={700} level="small">
            L
          </Text.Text>
        </Tooltip.Dialog>
      )}
      <Text.MaybeEditable
        id={`text-${key}`}
        value={name}
        onChange={onRename}
        allowDoubleClick={false}
      />
      <Ranger.TimeRangeChip level="small" timeRange={timeRange} />
      {labels.length > 0 && (
        <Flex.Box
          x
          gap="small"
          wrap
          style={{ overflowX: "auto", height: "fit-content" }}
        >
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

const Content = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  return (
    <Flex.Box empty full>
      <Toolbar.Header align="center" style={{ paddingRight: "0.5rem" }}>
        <Toolbar.Title icon={<Icon.Range />}>Ranges</Toolbar.Title>
        <Toolbar.Actions>
          {[
            {
              key: "create",
              children: <Icon.Add />,
              onClick: () => placeLayout(CREATE_LAYOUT),
              variant: "text",
            },
          ]}
        </Toolbar.Actions>
      </Toolbar.Header>
      <List />
    </Flex.Box>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "range",
  icon: <Icon.Range />,
  content: <Content />,
  tooltip: "Ranges",
  trigger: ["R"],
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
};
