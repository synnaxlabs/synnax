// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/status/Toolbar.css";

import { type status } from "@synnaxlabs/client";
import {
  Component,
  Flex,
  Icon,
  List as CoreList,
  Menu as PMenu,
  Select,
  Status,
  Tag,
  Telem,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useMemo } from "react";

import { EmptyAction, Toolbar } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { EXPLORER_LAYOUT } from "@/status/Explorer";
import { ContextMenu } from "@/status/list/ContextMenu";
import { useSelectFavorites } from "@/status/selectors";

const NoStatuses = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  return (
    <EmptyAction
      message="No favorite statuses."
      action="Open Status Explorer"
      onClick={() => placeLayout(EXPLORER_LAYOUT)}
    />
  );
};

const List = (): ReactElement => {
  const favorites = useSelectFavorites();
  const menuProps = PMenu.useContextMenu();

  const { getItem, subscribe, data } = Status.useList();
  const filteredData = useMemo(() => {
    const favoritesSet = new Set(favorites);
    return data.filter((key) => favoritesSet.has(key));
  }, [data, favorites]);

  return (
    <CoreList.Frame<status.Key, status.Status>
      data={filteredData}
      getItem={getItem}
      subscribe={subscribe}
    >
      <PMenu.ContextMenu
        menu={(p) => <ContextMenu {...p} getItem={getItem} />}
        {...menuProps}
      />
      <CoreList.Items<status.Key>
        full="y"
        emptyContent={<NoStatuses />}
        onContextMenu={menuProps.open}
      >
        {listItem}
      </CoreList.Items>
    </CoreList.Frame>
  );
};

const listItem = Component.renderProp((props: CoreList.ItemProps<status.Key>) => {
  const { itemKey } = props;
  const item = CoreList.useItem<status.Key, status.Status>(itemKey);
  if (item == null) return null;
  const { name, time, variant, message, labels } = item;
  return (
    <Select.ListItem className={CSS.B("status-list-item")} gap="small" y {...props}>
      <Flex.Box x justify="between">
        <Flex.Box x align="center" gap="small">
          <Status.Indicator variant={variant} />
          <Text.Text level="p" weight={450} status={variant}>
            {name}
          </Text.Text>
        </Flex.Box>
        <Telem.Text.TimeSpanSince level="small" format="semantic" variant="code">
          {time}
        </Telem.Text.TimeSpanSince>
      </Flex.Box>
      {message.length > 0 && <Text.Text level="small">{message}</Text.Text>}
      <Flex.Box x align="center" gap="small">
        {labels != null && labels.length > 0 && (
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
      </Flex.Box>
    </Select.ListItem>
  );
});

const Content = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  return (
    <Ontology.Toolbar>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.Status />}>Statuses</Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.Action
            tooltip="Open Status Explorer"
            onClick={() => placeLayout(EXPLORER_LAYOUT)}
            variant="filled"
          >
            <Icon.Explore />
          </Toolbar.Action>
        </Toolbar.Actions>
      </Toolbar.Header>
      <List />
    </Ontology.Toolbar>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "status",
  icon: <Icon.Status />,
  content: <Content />,
  tooltip: "Statuses",
  trigger: ["S"],
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
};
