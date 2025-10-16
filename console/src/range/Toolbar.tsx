// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Toolbar.css";

import {
  Component,
  ContextMenu as PContextMenu,
  Flex,
  type Flux,
  Haul,
  Icon,
  List as CoreList,
  Ranger,
  Select,
  Tag,
  Telem,
  Text,
  Tooltip,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import { EmptyAction, Toolbar } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { ContextMenu } from "@/range/ContextMenu";
import { CREATE_LAYOUT } from "@/range/Create";
import { EXPLORER_LAYOUT } from "@/range/Explorer";
import { select, useSelect, useSelectStaticKeys } from "@/range/selectors";
import { add, rename, setActive, type StaticRange } from "@/range/slice";
import { type RootState } from "@/store";

const NoRanges = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  const handleLinkClick = () => placeLayout(CREATE_LAYOUT);
  return (
    <EmptyAction
      message="No ranges loaded."
      action="Create a range"
      onClick={handleLinkClick}
    />
  );
};

const List = (): ReactElement => {
  const dispatch = useDispatch();
  const activeRange = useSelect();
  const data = useSelectStaticKeys();

  const handleSelect = (key: string): void => {
    dispatch(setActive(key));
  };

  const dropProps = Haul.useDrop({
    type: "range-toolbar",
    canDrop: Haul.canDropOfType("range"),
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

  const contextMenuProps = PContextMenu.use();

  return (
    <Select.Frame<string, StaticRange>
      data={data}
      value={activeRange?.key}
      onChange={handleSelect}
    >
      <PContextMenu.ContextMenu
        menu={(p) => <ContextMenu {...p} />}
        {...contextMenuProps}
      />
      <CoreList.Items
        full="y"
        emptyContent={<NoRanges />}
        {...dropProps}
        onContextMenu={contextMenuProps.open}
      >
        {listItem}
      </CoreList.Items>
    </Select.Frame>
  );
};

export const useRename = () => {
  const store = useStore<RootState>();
  return Ranger.useRename({
    beforeUpdate: useCallback(
      async ({ data, rollbacks }: Flux.BeforeUpdateParams<Ranger.RenameParams>) => {
        const { key, name } = data;
        const rng = select(store.getState(), key);
        if (rng == null) return data;
        const oldName = rng.name;
        if (!rng.persisted) return false;
        store.dispatch(rename({ key, name }));
        rollbacks.push(() => store.dispatch(rename({ key, name: oldName })));
        return data;
      },
      [store],
    ),
  });
};

const listItem = Component.renderProp((props: CoreList.ItemProps<string>) => {
  const { itemKey } = props;
  const entry = useSelect(itemKey);
  const labels = Ranger.useLabels(itemKey)?.data ?? [];
  const onRename = useRename();
  if (entry == null || entry.variant === "dynamic") return null;
  const { key, name, timeRange, persisted } = entry;
  return (
    <Select.ListItem className={CSS.B("range-list-item")} {...props} gap="small" y>
      {!persisted && (
        <Tooltip.Dialog location="left">
          <Text.Text level="small">This range is local.</Text.Text>
          <Text.Text className="save-button" weight={700} level="small" color={11}>
            L
          </Text.Text>
        </Tooltip.Dialog>
      )}
      <Flex.Box x align="center" gap="small">
        <Ranger.StageIcon timeRange={timeRange} />
        <Text.MaybeEditable
          id={`text-${key}`}
          level="p"
          value={name}
          onChange={(name) => onRename.update({ key, name })}
          allowDoubleClick={false}
        />
      </Flex.Box>
      <Telem.Text.TimeRange level="small">{timeRange}</Telem.Text.TimeRange>
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
    <Ontology.Toolbar>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.Range />}>Ranges</Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.Action
            tooltip="Create Range"
            onClick={() => placeLayout(CREATE_LAYOUT)}
          >
            <Icon.Add />
          </Toolbar.Action>
          <Toolbar.Action
            tooltip="Open Range Explorer"
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
  key: "range",
  icon: <Icon.Range />,
  content: <Content />,
  tooltip: "Ranges",
  trigger: ["R"],
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
};
