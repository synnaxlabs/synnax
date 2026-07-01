// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/service/range/Toolbar.css";

import { ranger } from "@synnaxlabs/client";
import {
  Access,
  Component,
  Flex,
  type Flux,
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
import { type ReactElement, useCallback } from "react";

import { CSS } from "@/component/css";
import { Empty } from "@/component/empty";
import { type Nav } from "@/component/nav";
import { useCreateModal } from "@/component/range/useCreateModal";
import { Toolbar } from "@/component/toolbar";
import { ContextMenu } from "@/service/range/ContextMenu";
import { EXPLORER_LAYOUT } from "@/service/range/Explorer";
import { Session } from "@/session";
import {
  selectState as selectRange,
  useSelectState,
  useSelectStaticKeys,
} from "@/session/range/selectors";
import { add, rename, select, type StaticState } from "@/session/range/slice";

const NoRanges = (): ReactElement => {
  const placeLayout = Session.Layout.usePlacer();
  const hasRetrievePermission = Access.useRetrieveGranted(ranger.TYPE_ONTOLOGY_ID);
  return (
    <Empty.Action
      message="No favorited ranges."
      action={hasRetrievePermission ? "Open Range Explorer" : undefined}
      onClick={() => placeLayout(EXPLORER_LAYOUT)}
    />
  );
};

const List = (): ReactElement => {
  const dispatch = Session.useDispatch();
  const activeRange = useSelectState();
  const data = useSelectStaticKeys();

  const handleSelect = (key: string): void => {
    dispatch(select(key));
  };

  const dropProps = Haul.useDrop({
    type: "range_toolbar",
    canDrop: Ranger.canDropHaulItem,
    onDrop: ({ items }) => {
      const dropped = Ranger.filterHaulItems(items);
      dropped.forEach(({ data }) =>
        dispatch(add({ ...data, persisted: true, variant: "static" })),
      );
      return dropped;
    },
  });

  const menuProps = Menu.useContextMenu();

  return (
    <Select.Frame<string, StaticState>
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

export const useRename = () => {
  const store = Session.useStore();
  return Ranger.useRename({
    beforeUpdate: useCallback(
      async ({ data, rollbacks }: Flux.BeforeUpdateParams<Ranger.RenameParams>) => {
        const { key, name } = data;
        const rng = selectRange(store.getState(), key);
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

const listItem = Component.renderProp((props: BaseList.ItemProps<string>) => {
  const { itemKey } = props;
  const entry = useSelectState(itemKey);
  const isLocal = entry != null && !entry.persisted;
  const labels =
    Ranger.useLabels(itemKey, {
      beforeRetrieve: useCallback(() => (isLocal ? [] : true), [isLocal]),
    })?.data ?? [];
  const onRename = useRename();
  const hasUpdatePermission = Access.useUpdateGranted(ranger.ontologyID(itemKey));
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
          onChange={
            hasUpdatePermission ? (name) => onRename.update({ key, name }) : undefined
          }
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

const Actions = (): ReactElement | null => {
  const placeLayout = Session.Layout.usePlacer();
  const openCreate = useCreateModal();
  const hasCreatePermission = Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID);
  const hasRetrievePermission = Access.useRetrieveGranted(ranger.TYPE_ONTOLOGY_ID);
  if (!hasCreatePermission && !hasRetrievePermission) return null;
  return (
    <Toolbar.Actions>
      {hasCreatePermission && (
        <Toolbar.Action tooltip="Create range" onClick={() => openCreate()}>
          <Icon.Add />
        </Toolbar.Action>
      )}
      {hasRetrievePermission && (
        <Toolbar.Action
          tooltip="Open Range Explorer"
          onClick={() => placeLayout(EXPLORER_LAYOUT)}
          variant="filled"
        >
          <Icon.Explore />
        </Toolbar.Action>
      )}
    </Toolbar.Actions>
  );
};

const Content = (): ReactElement => (
  <Toolbar.Content>
    <Toolbar.Header padded>
      <Toolbar.Title icon={<Icon.Range />}>Ranges</Toolbar.Title>
      <Actions />
    </Toolbar.Header>
    <List />
  </Toolbar.Content>
);

export const TOOLBAR: Nav.Item = {
  key: "range",
  icon: <Icon.Range />,
  content: <Content />,
  tooltip: "Ranges",
  trigger: ["R"],
  initialSize: 300,
  sizeBounds: { lower: 175, upper: 400 },
  useVisible: () => Access.useRetrieveGranted(ranger.TYPE_ONTOLOGY_ID),
};
