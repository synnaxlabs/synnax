// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/toolbar/Symbols.css";

import { group, type ontology, type schematic } from "@synnaxlabs/client";
import {
  Button,
  Component,
  Flex,
  Group,
  Haul,
  Icon,
  Input,
  List,
  Schematic,
  Select,
  Status,
  Symbol,
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { useSelectSelectedSymbolGroup } from "@/schematic/selectors";
import { addElement, setSelectedSymbolGroup } from "@/schematic/slice";
import { createCreateLayout } from "@/schematic/symbols/Create";

const StaticListItem = (props: List.ItemProps<string>): ReactElement | null => {
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

const staticListItem = Component.renderProp(StaticListItem);

export interface GroupProps
  extends Pick<
    List.FrameProps<string, Schematic.Spec>,
    "data" | "getItem" | "subscribe"
  > {
  onSelect: (key: string) => void;
}

export interface StaticGroupProps extends Pick<GroupProps, "onSelect"> {
  groupKey: group.Key;
}

const StaticSymbolList = ({ groupKey, onSelect }: StaticGroupProps): ReactElement => {
  const group = Schematic.SYMBOL_GROUPS.find((g) => g.key === groupKey);
  const symbols = useMemo(
    () =>
      Object.values(Schematic.SYMBOLS).filter((s) => group?.symbols.includes(s.key)),
    [group],
  );
  const { data, getItem } = List.useStaticData<string, Schematic.Spec>({
    data: symbols,
  });
  return (
    <Select.Frame<string, Schematic.Spec>
      data={data}
      getItem={getItem}
      value={undefined}
      allowNone
      onChange={onSelect}
    >
      <List.Items x className={CSS.BE("schematic", "symbols", "group")} wrap>
        {staticListItem}
      </List.Items>
    </Select.Frame>
  );
};

export interface RemoteListItemProps extends List.ItemProps<string> {
  itemKey: string;
}
const RemoteListItem = (props: RemoteListItemProps): ReactElement | null => {
  const { itemKey } = props;
  const symbol = List.useItem<string, schematic.symbol.Symbol>(itemKey);
  if (symbol == null) return null;
  return (
    <Select.ListItem {...props} y level="small" justify="center">
      {symbol.name}
    </Select.ListItem>
  );
};

const remoteListItem = Component.renderProp(RemoteListItem);

const RemoteSymbolList = ({ groupKey, onSelect }: StaticGroupProps): ReactElement => {
  const listData = Symbol.useList({
    initialParams: { parent: group.ontologyID(groupKey) },
  });
  const { fetchMore } = List.usePager({ retrieve: listData.retrieve });
  useEffect(() => fetchMore(), [fetchMore]);
  return (
    <Select.Frame<string, schematic.symbol.Symbol>
      {...listData}
      value={undefined}
      allowNone
      onChange={onSelect}
    >
      <List.Items x className={CSS.BE("schematic", "symbols", "group")} wrap>
        {remoteListItem}
      </List.Items>
    </Select.Frame>
  );
};

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

export interface ActionsProps {
  symbolGroupID: ontology.ID;
  selectedGroup: string;
}

const Actions = ({ symbolGroupID, selectedGroup }: ActionsProps): ReactElement => {
  const { updateAsync } = Group.create.useDirect({ params: {} });
  const rename = Modals.useRename();
  const handleError = Status.useErrorHandler();
  const placeLayout = Layout.usePlacer();
  const handleCreateGroup = useCallback(() => {
    handleError(async () => {
      const result = await rename(
        {
          initialValue: "New Group",
          allowEmpty: false,
          label: "Group Name",
        },
        {
          name: "Schematic.Symbols.Create Group",
          icon: "Group",
        },
      );
      if (result == null) return;
      await updateAsync({
        key: uuid.create(),
        name: result,
        parent: symbolGroupID,
      });
    }, "Failed to create group");
  }, [updateAsync, rename, handleError, symbolGroupID]);

  const isRemoteGroup = group.keyZ.safeParse(selectedGroup).success;

  const handleCreateSymbol = useCallback(() => {
    if (!isRemoteGroup) return;
    placeLayout(
      createCreateLayout({
        args: { parent: group.ontologyID(selectedGroup) },
      }),
    );
  }, [isRemoteGroup, placeLayout, selectedGroup]);

  return (
    <Flex.Box y className={CSS.BE("schematic", "symbols", "actions")}>
      <Button.Button variant="outlined" size="medium" tooltip="Create new symbol group">
        <CreateGroupIcon onClick={handleCreateGroup} />
      </Button.Button>
      <Button.Button
        variant="outlined"
        size="medium"
        tooltip="Create new symbol"
        disabled={!isRemoteGroup}
        onClick={handleCreateSymbol}
      >
        <CreateSymbolIcon />
      </Button.Button>
    </Flex.Box>
  );
};

export interface GroupListProps extends Input.Control<group.Key> {
  symbolGroupID: ontology.ID;
}

const GroupList = ({
  value,
  onChange,
  symbolGroupID,
}: GroupListProps): ReactElement => {
  const staticData = List.useStaticData<group.Key, group.Payload>({
    data: Schematic.SYMBOL_GROUPS,
  });
  const remoteData = Group.useList({ initialParams: { parent: symbolGroupID } });
  const data = List.useCombinedData({
    first: staticData,
    second: remoteData,
  });
  const { fetchMore } = List.usePager({ retrieve: remoteData.retrieve });
  useEffect(() => {
    fetchMore();
  }, [fetchMore]);
  return (
    <Select.Frame<group.Key, group.Payload> {...data} value={value} onChange={onChange}>
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
  const isRemoteGroup = group.keyZ.safeParse(groupKey).success;
  const handleAddElement = useCallback(
    (key: string) => {
      let variant: Schematic.Variant;
      if (isRemoteGroup) variant = "actuator";
      else variant = key as Schematic.Variant;
      const spec = Schematic.SYMBOLS[variant];
      const initialProps = spec.defaultProps(theme);
      dispatch(
        addElement({
          key: layoutKey,
          elKey: id.create(),
          node: { zIndex: spec.zIndex },
          props: { key: variant, ...initialProps },
        }),
      );
    },
    [dispatch, layoutKey, theme, isRemoteGroup],
  );

  const [search, setSearch] = useState("");
  const g = Symbol.useGroup.useDirect({ params: {} });
  return (
    <Flex.Box x empty className={CSS.BE("schematic", "symbols")}>
      {g.data != null && (
        <Actions
          symbolGroupID={group.ontologyID(g.data.key)}
          selectedGroup={groupKey}
        />
      )}
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
        {g.data != null && (
          <GroupList
            value={groupKey}
            onChange={setGroupKey}
            symbolGroupID={group.ontologyID(g.data.key)}
          />
        )}
      </Flex.Box>
      {isRemoteGroup ? (
        <RemoteSymbolList groupKey={groupKey} onSelect={handleAddElement} />
      ) : (
        <StaticSymbolList groupKey={groupKey} onSelect={handleAddElement} />
      )}
    </Flex.Box>
  );
};
