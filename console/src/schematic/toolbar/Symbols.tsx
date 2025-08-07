// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/toolbar/Symbols.css";

import { type group } from "@synnaxlabs/client";
import {
  Button,
  Component,
  Flex,
  Haul,
  Icon,
  Input,
  List,
  Schematic,
  Select,
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { useSelectSelectedSymbolGroup } from "@/schematic/selectors";
import { addElement, setSelectedSymbolGroup } from "@/schematic/slice";

const ListItem = (props: List.ItemProps<string>): ReactElement | null => {
  const { itemKey } = props;
  const theme = Theming.use();

  const { startDrag, onDragEnd } = Haul.useDrag({
    type: "Diagram-Elements",
    key: "symbols",
  });

  const handleDragStart = useCallback(() => {
    startDrag([{ type: "schematic-element", key: itemKey }]);
  }, [startDrag, itemKey]);
  const spec = List.useItem<string, Schematic.Spec>(itemKey);
  const defaultProps_ = useMemo(() => spec?.defaultProps(theme), [spec, theme]);
  if (spec == null || defaultProps_ == null) return null;
  const { name, Preview } = spec;
  return (
    <Select.ListItem
      className={CSS(CSS.BE("schematic-symbols", "button"))}
      align="center"
      gap="tiny"
      draggable
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
      y
      {...props}
    >
      <Text.Text level="small">{name}</Text.Text>
      <Flex.Box align="center" justify="center" grow>
        <Preview {...defaultProps_} scale={0.75} />
      </Flex.Box>
    </Select.ListItem>
  );
};

const listItem = Component.renderProp(ListItem);

export interface GroupProps
  extends Pick<
    List.FrameProps<string, Schematic.Spec>,
    "data" | "getItem" | "subscribe"
  > {
  onSelect: (key: string) => void;
}

const BaseGroup = ({
  data,
  getItem,
  subscribe,
  onSelect,
}: GroupProps): ReactElement => (
  <Select.Frame<string, Schematic.Spec>
    data={data}
    getItem={getItem}
    subscribe={subscribe}
    value={undefined}
    allowNone
    onChange={onSelect}
  >
    <List.Items x className={CSS.BE("schematic", "symbols", "group")} wrap>
      {listItem}
    </List.Items>
  </Select.Frame>
);

export interface StaticGroupProps extends Pick<GroupProps, "onSelect"> {
  groupKey: group.Key;
}

const StaticGroup = ({ groupKey, onSelect }: StaticGroupProps): ReactElement => {
  const group = Schematic.SYMBOL_GROUPS.find((g) => g.key === groupKey);
  const symbols = useMemo(
    () =>
      Object.values(Schematic.SYMBOLS).filter((s) => group?.symbols.includes(s.key)),
    [group],
  );
  const data = List.useStaticData<string, Schematic.Spec>({
    data: symbols,
  });
  return <BaseGroup {...data} onSelect={onSelect} />;
};

export interface GroupListProps extends Input.Control<group.Key> {}

const GroupListItem = (props: List.ItemProps<group.Key>): ReactElement | null => {
  const { itemKey } = props;
  const group = List.useItem<group.Key, group.Payload>(itemKey);
  if (group == null) return null;
  return (
    <Select.ListItem {...props} y level="small" justify="center">
      {group.name}
    </Select.ListItem>
  );
};

const groupListItem = Component.renderProp(GroupListItem);

const CreateGroupIcon = Icon.createComposite(Icon.Group, {
  bottomRight: Icon.Add,
});

const CreateSymbolIcon = Icon.createComposite(Icon.Schematic, {
  bottomRight: Icon.Add,
});

const Actions = (): ReactElement => (
  <Flex.Box y className={CSS.BE("schematic", "symbols", "actions")}>
    <Button.Button variant="outlined" size="medium" tooltip="Create new symbol group">
      <CreateGroupIcon />
    </Button.Button>
    <Button.Button variant="outlined" size="medium" tooltip="Create new symbol">
      <CreateSymbolIcon />
    </Button.Button>
  </Flex.Box>
);

const GroupList = ({ value, onChange }: GroupListProps): ReactElement => {
  const staticData = List.useStaticData<group.Key, group.Payload>({
    data: Schematic.SYMBOL_GROUPS,
  });
  return (
    <Select.Frame<group.Key, group.Payload>
      {...staticData}
      value={value}
      onChange={onChange}
    >
      <List.Items>{groupListItem}</List.Items>
    </Select.Frame>
  );
};

export const Symbols = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const theme = Theming.use();
  const dispatch = useDispatch();
  const groupKey = useSelectSelectedSymbolGroup();
  const setGroupKey = useCallback(
    (group: group.Key) => {
      dispatch(setSelectedSymbolGroup({ group }));
    },
    [dispatch],
  );
  const handleAddElement = useCallback(
    (key: string) => {
      const spec = Schematic.SYMBOLS[key as Schematic.Variant];
      const initialProps = spec.defaultProps(theme);
      dispatch(
        addElement({
          key: layoutKey,
          elKey: id.create(),
          node: { zIndex: spec.zIndex },
          props: { key, ...initialProps },
        }),
      );
    },
    [dispatch, layoutKey, theme],
  );

  const [search, setSearch] = useState("");

  return (
    <Flex.Box x empty className={CSS.BE("schematic", "symbols")}>
      <Actions />
      <Flex.Box
        pack
        y
        sharp
        className={CSS.BE("schematic", "symbols", "group", "list")}
      >
        <Input.Text
          value={search}
          onChange={setSearch}
          placeholder="Search Symbols"
          size="small"
        />
        <GroupList value={groupKey} onChange={setGroupKey} />
      </Flex.Box>
      <StaticGroup groupKey={groupKey} onSelect={handleAddElement} />
    </Flex.Box>
  );
};
