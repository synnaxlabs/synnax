// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Toolbar.css";

import { ranger } from "@synnaxlabs/client";
import {
  Align,
  Component,
  Haul,
  Icon,
  List as CoreList,
  Menu as PMenu,
  Ranger,
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
import { select, useSelect, useSelectMultiple } from "@/range/selectors";
import { add, rename, setActive, type StaticRange } from "@/range/slice";
import { type RootState } from "@/store";

const NoRanges = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  const handleLinkClick = () => {
    placeLayout(CREATE_LAYOUT);
  };
  return (
    <Align.Space
      empty
      style={{ height: "100%", position: "relative", padding: "1rem" }}
      className={CSS.B("range-toolbar-no-ranges")}
    >
      <Align.Center y style={{ height: "100%" }} size="medium">
        <Text.Text level="p">No ranges loaded.</Text.Text>
        <Text.Link level="p" onClick={handleLinkClick}>
          Create a Range
        </Text.Link>
      </Align.Center>
    </Align.Space>
  );
};

const List = (): ReactElement => {
  const dispatch = useDispatch();
  const activeRange = useSelect();
  const ranges = useSelectMultiple();

  const handleSelect = (key: string): void => {
    dispatch(setActive(key));
  };

  const drop = Haul.useDrop({
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
    <CoreList.List<string, StaticRange>
      data={ranges.filter((r) => r.variant === "static")}
      emptyContent={<NoRanges />}
    >
      <PMenu.ContextMenu menu={(p) => <ContextMenu {...p} />} {...menuProps}>
        <CoreList.Selector
          value={activeRange?.key ?? null}
          onChange={handleSelect}
          allowMultiple={false}
          allowNone={true}
        >
          <CoreList.Core
            style={{ height: "100%", overflowX: "hidden" }}
            onContextMenu={menuProps.open}
            className={menuProps.className}
            {...drop}
          >
            {Component.renderProp(ListItem)}
          </CoreList.Core>
        </CoreList.Selector>
      </PMenu.ContextMenu>
    </CoreList.List>
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
    client?.ranges
      .rename(key, name)
      .catch((e) => handleError(e, "Failed to rename range"));
  };
};

interface ListItemProps extends CoreList.ItemProps<string, StaticRange> {}

const ListItem = (props: ListItemProps): ReactElement => {
  const { entry } = props;
  const labels = Ranger.useLabels(entry.key)?.data ?? [];
  const onRename = useRename(entry.key);

  return (
    <CoreList.ItemFrame className={CSS.B("range-list-item")} {...props} size="small" y>
      {!entry.persisted && (
        <Tooltip.Dialog location="left">
          <Text.Text level="small">This range is local.</Text.Text>
          <Text.Text className="save-button" weight={700} level="small" shade={11}>
            L
          </Text.Text>
        </Tooltip.Dialog>
      )}
      <Text.MaybeEditable
        id={`text-${entry.key}`}
        level="p"
        value={entry.name}
        onChange={onRename}
        allowDoubleClick={false}
      />
      <Ranger.TimeRangeChip level="small" timeRange={entry.timeRange} />
      {labels.length > 0 && (
        <Align.Space
          x
          size="small"
          wrap
          style={{ overflowX: "auto", height: "fit-content" }}
        >
          {labels.map((l) => (
            <Tag.Tag key={l.key} size="tiny" color={l.color}>
              {l.name}
            </Tag.Tag>
          ))}
        </Align.Space>
      )}
    </CoreList.ItemFrame>
  );
};

const Content = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  return (
    <Align.Space empty style={{ height: "100%" }}>
      <Toolbar.Header align="center" style={{ paddingRight: "0.5rem" }}>
        <Toolbar.Title icon={<Icon.Range />}>Ranges</Toolbar.Title>
        <Toolbar.Actions>
          {[
            {
              key: "create",
              children: <Icon.Add />,
              onClick: () => placeLayout(CREATE_LAYOUT),
            },
          ]}
        </Toolbar.Actions>
      </Toolbar.Header>
      <List />
    </Align.Space>
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
