// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/editor/graph/toolbar/Stages.css";

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

import { useAddSymbol } from "@/arc/useAddSymbol";
import { CSS } from "@/css";

const StaticListItem = (props: List.ItemProps<string>): ReactElement | null => {
  const { itemKey } = props;
  const { startDrag, onDragEnd } = Haul.useDrag({
    type: "diagram_elements",
    key: "stages",
  });
  const theme = Theming.use();
  const handleDragStart = useCallback(() => {
    startDrag([Arc.createHaulItem(itemKey)]);
  }, [startDrag, itemKey]);

  const spec = List.useItem<string, Arc.Graph.Node.Spec>(itemKey);
  const config = useMemo(() => spec?.defaultConfig(theme), [spec, theme]);
  if (spec == null || config == null) return null;
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
        <Preview config={config} scale={0.75} />
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
  const stages = useMemo<Arc.Graph.Node.Spec[]>(() => {
    const g = Arc.Graph.Node.GROUPS.find((g) => g.key === groupKey);
    return Object.values(Arc.Graph.Node.REGISTRY).filter((s) =>
      g?.symbols.includes(s.key),
    ) as unknown as Arc.Graph.Node.Spec[];
  }, [groupKey]);
  const { data, getItem } = List.useStaticData<string, Arc.Graph.Node.Spec>({
    data: stages,
  });
  return (
    <Select.Frame<string, Arc.Graph.Node.Spec>
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
  const group = List.useItem<string, Arc.Graph.Node.Group>(itemKey);
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
  const { data, getItem } = List.useStaticData<string, Arc.Graph.Node.Group>({
    data: Arc.Graph.Node.GROUPS,
  });
  return (
    <Select.Frame<string, Arc.Graph.Node.Group>
      data={data}
      getItem={getItem}
      value={value}
      onChange={onChange}
    >
      <List.Items<string, Arc.Graph.Node.Group> x gap="small">
        {groupListItem}
      </List.Items>
    </Select.Frame>
  );
};

export const Stages = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const [selectedGroup, setSelectedGroup] = useState<string>("basic");
  const addSymbol = useAddSymbol(layoutKey);
  return (
    <Flex.Box y empty full className={CSS.BE("arc", "stages")}>
      <Flex.Box x sharp className={CSS.BE("arc", "stages", "group", "list")}>
        <GroupList value={selectedGroup} onChange={setSelectedGroup} />
      </Flex.Box>
      <StaticStageList groupKey={selectedGroup} onSelect={addSymbol} />
    </Flex.Box>
  );
};
