// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/editor/toolbar/Stages.css";

import {
  Arc,
  Button,
  Component,
  Flex,
  Haul,
  type Input,
  List,
  Menu,
  Select,
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { useAddSymbol } from "@/arc/useAddSymbol";
import { CSS } from "@/css";

const StaticListItem = (props: List.ItemProps<string>): ReactElement | null => {
  const { itemKey } = props;
  const { startDrag, onDragEnd } = Haul.useDrag({
    type: "Diagram-Elements",
    key: "stages",
  });
  const theme = Theming.use();
  const handleDragStart = useCallback(() => {
    startDrag([{ type: "arc-element", key: itemKey }]);
  }, [startDrag, itemKey]);

  const spec = List.useItem<string, Arc.Stage.Spec>(itemKey);
  const defaultProps_ = useMemo(() => spec?.defaultProps(theme), [spec, theme]);
  if (spec == null || defaultProps_ == null) return null;
  const { name, Preview } = spec;

  return (
    <Select.ListItem
      className={CSS(CSS.BE("arc-stages", "button"))}
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

export const staticListItem = Component.renderProp(StaticListItem);

export interface StateListProps {
  groupKey: string;
  onSelect: (key: string) => void;
}

export const StaticStageList = ({
  groupKey,
  onSelect,
}: StateListProps): ReactElement => {
  const stages = useMemo(() => {
    const g = Arc.Stage.GROUPS.find((g) => g.key === groupKey);
    return Object.values(Arc.Stage.REGISTRY).filter((s) => g?.symbols.includes(s.key));
  }, [groupKey]);
  const { data, getItem } = List.useStaticData<string, Arc.Stage.Spec>({
    data: stages,
  });
  return (
    <Select.Frame<string, Arc.Stage.Spec>
      data={data}
      getItem={getItem}
      value={undefined}
      allowNone
      onChange={onSelect}
    >
      <List.Items x className={CSS.BE("arc", "stages", "group")} wrap>
        {staticListItem}
      </List.Items>
    </Select.Frame>
  );
};

const groupListItem = Component.renderProp((props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const group = List.useItem<string, Arc.Stage.Group>(itemKey);
  const { selected, onSelect } = Select.useItemState(itemKey);
  if (group == null) return null;
  const { Icon, name } = group;
  return (
    <Button.Toggle
      id={itemKey.toString()}
      size="small"
      value={selected}
      onChange={onSelect}
      className={CSS(Menu.CONTEXT_TARGET, selected && Menu.CONTEXT_SELECTED)}
      textColor={9}
    >
      <Icon />
      {name}
    </Button.Toggle>
  );
});

export interface GroupListProps extends Input.Control<string> {}

const GroupList = ({ value, onChange }: GroupListProps) => {
  const { data, getItem } = List.useStaticData<string, Arc.Stage.Group>({
    data: Arc.Stage.GROUPS,
  });
  return (
    <Select.Frame<string, Arc.Stage.Group>
      data={data}
      getItem={getItem}
      value={value}
      onChange={onChange}
    >
      <List.Items<string, Arc.Stage.Group> x gap="small">
        {groupListItem}
      </List.Items>
    </Select.Frame>
  );
};

export const Stages = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const [selectedGroup, setSelectedGroup] = useState<string>("basic");
  const dispatch = useDispatch();
  const addSymbol = useAddSymbol(dispatch, layoutKey);
  return (
    <Flex.Box y empty full className={CSS.BE("arc", "stages")}>
      <Flex.Box x sharp className={CSS.BE("arc", "stages", "group", "list")}>
        <GroupList value={selectedGroup} onChange={setSelectedGroup} />
      </Flex.Box>
      <StaticStageList groupKey={selectedGroup} onSelect={addSymbol} />
    </Flex.Box>
  );
};
