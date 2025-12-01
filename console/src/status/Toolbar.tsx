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
import { type ReactElement, useEffect, useState } from "react";
import { useDispatch } from "react-redux";

import { EmptyAction, Toolbar } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { CREATE_LAYOUT } from "@/status/Create";
import { EXPLORER_LAYOUT } from "@/status/Explorer";
import { contextMenuRenderProp } from "@/status/list/ContextMenu";
import { useSelectFavorites } from "@/status/selectors";
import { removeFavorites } from "@/status/slice";

const NoStatuses = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  return (
    <EmptyAction
      message="No favorited statuses."
      action="Open Status Explorer"
      onClick={() => placeLayout(EXPLORER_LAYOUT)}
    />
  );
};

const List = (): ReactElement => {
  const favorites = useSelectFavorites();
  const menuProps = PMenu.useContextMenu();
  const [selected, setSelected] = useState<status.Key[]>([]);
  return (
    <Select.Frame<status.Key, status.Status>
      multiple
      data={favorites}
      value={selected}
      onChange={setSelected}
    >
      <PMenu.ContextMenu menu={contextMenuRenderProp} {...menuProps} />
      <CoreList.Items<status.Key>
        full="y"
        emptyContent={<NoStatuses />}
        onContextMenu={menuProps.open}
      >
        {listItem}
      </CoreList.Items>
    </Select.Frame>
  );
};

const ListItem = (props: CoreList.ItemProps<status.Key>) => {
  const { itemKey } = props;
  const q = Status.useRetrieve({ key: itemKey });
  const dispatch = useDispatch();
  useEffect(() => {
    if (q.variant === "error") dispatch(removeFavorites([itemKey]));
  }, [q.variant, dispatch, itemKey]);
  if (q.variant !== "success") return null;
  const item = q.data;
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
        <Telem.Text.TimeSpanSince
          level="small"
          format="semantic"
          variant="code"
          color={9}
        >
          {time}
        </Telem.Text.TimeSpanSince>
      </Flex.Box>
      {message.length > 0 && (
        <Text.Text level="small" color={9}>
          {message}
        </Text.Text>
      )}
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
    </Select.ListItem>
  );
};

const listItem = Component.renderProp(ListItem);

const Content = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  return (
    <Toolbar.Content>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.Status />}>Statuses</Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.Action
            tooltip="Create status"
            onClick={() => placeLayout(CREATE_LAYOUT)}
          >
            <Icon.Add />
          </Toolbar.Action>
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
    </Toolbar.Content>
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
