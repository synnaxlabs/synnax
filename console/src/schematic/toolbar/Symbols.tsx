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
  Menu,
  Schematic,
  SchematicSymbol,
  Select,
  Status,
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { EmptyAction } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";
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
  const Preview = Schematic.SYMBOLS.actuator.Preview;

  const { startDrag, onDragEnd } = Haul.useDrag({
    type: "Diagram-Elements",
    key: "symbols",
  });

  const handleDragStart = useCallback(() => {
    startDrag([
      { type: "schematic-element", key: "actuator", data: { specKey: itemKey } },
    ]);
  }, [startDrag, itemKey]);

  if (symbol == null) return null;

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
      <Text.Text level="small">{symbol.name}</Text.Text>
      <Flex.Box align="center" justify="center" grow>
        <Preview specKey={itemKey} scale={0.75} />
      </Flex.Box>
    </Select.ListItem>
  );
};

const remoteListItem = Component.renderProp(RemoteListItem);

export interface RemoteSymbolListContextMenuProps extends Menu.ContextMenuMenuProps {
  groupKey: string;
}

const RemoteSymbolListContextMenu = (
  props: RemoteSymbolListContextMenuProps,
): ReactElement => {
  const firstKey = props.keys[0];
  const item = List.useItem<schematic.symbol.Key, schematic.symbol.Symbol>(firstKey);
  const confirmDelete = useConfirmDelete({
    type: "Schematic.Symbol",
    icon: "Schematic",
  });
  const placeLayout = Layout.usePlacer();
  const renameModal = Modals.useRename();
  const rename = SchematicSymbol.useRename({
    params: { key: firstKey },
    beforeUpdate: async ({ value }) => {
      if (item == null) return false;
      const newName = await renameModal(
        {
          initialValue: value,
          allowEmpty: false,
          label: "Symbol Name",
        },
        {
          name: "Schematic.Symbols.Rename",
          icon: "Schematic",
        },
      );
      if (newName == null) return false;
      return newName;
    },
  });
  const del = SchematicSymbol.useDelete({
    params: { key: firstKey },
    beforeUpdate: async () => {
      if (item == null) return false;
      return await confirmDelete({ name: item.name });
    },
  });
  const handleEdit = () => {
    placeLayout(
      createCreateLayout({
        args: { key: firstKey, parent: group.ontologyID(props.groupKey) },
      }),
    );
  };
  const handleSelect: Menu.MenuProps["onChange"] = {
    delete: () => del.update(),
    rename: () => rename.update(item?.name ?? ""),
    edit: handleEdit,
  };
  return (
    <Menu.Menu level="small" gap="small" onChange={handleSelect}>
      <Menu.Item itemKey="delete">
        <Icon.Delete />
        Delete
      </Menu.Item>
      <Menu.Item itemKey="rename">
        <Icon.Rename />
        Rename
      </Menu.Item>
      <Menu.Item itemKey="edit">
        <Icon.Edit />
        Edit
      </Menu.Item>
    </Menu.Menu>
  );
};

const useCreateSymbol = (selectedGroup: string) => {
  const placeLayout = Layout.usePlacer();
  const handleCreateSymbol = useCallback(() => {
    placeLayout(
      createCreateLayout({
        args: { parent: group.ontologyID(selectedGroup) },
      }),
    );
  }, [placeLayout, selectedGroup]);
  return handleCreateSymbol;
};

interface RemoteListEmptyContentProps {
  groupKey: string;
}

const RemoteListEmptyContent = ({
  groupKey,
}: RemoteListEmptyContentProps): ReactElement => {
  const createSymbol = useCreateSymbol(groupKey);
  return (
    <EmptyAction
      message="No symbols found"
      action="Create Symbol"
      onClick={createSymbol}
    />
  );
};

const RemoteSymbolList = ({ groupKey, onSelect }: StaticGroupProps): ReactElement => {
  const listData = SchematicSymbol.useList({
    initialParams: { parent: group.ontologyID(groupKey) },
  });
  const { fetchMore } = List.usePager({ retrieve: listData.retrieve });
  useEffect(() => fetchMore(), [fetchMore]);
  const menuProps = Menu.useContextMenu();
  return (
    <Select.Frame<string, schematic.symbol.Symbol>
      {...listData}
      value={undefined}
      allowNone
      onChange={onSelect}
    >
      <Menu.ContextMenu
        {...menuProps}
        menu={(props) => <RemoteSymbolListContextMenu {...props} groupKey={groupKey} />}
      >
        <List.Items
          x
          className={CSS.BE("schematic", "symbols", "group")}
          onContextMenu={menuProps.open}
          emptyContent={<RemoteListEmptyContent groupKey={groupKey} />}
          wrap
        >
          {remoteListItem}
        </List.Items>
      </Menu.ContextMenu>
    </Select.Frame>
  );
};

const GroupListItem = (props: List.ItemProps<group.Key>): ReactElement | null => {
  const { itemKey } = props;
  const group = List.useItem<group.Key, group.Payload & { Icon?: Icon.FC }>(itemKey);
  const { selected, onSelect } = Select.useItemState(itemKey);
  if (group == null) return null;
  const { Icon: GroupIcon } = group;
  return (
    <Button.Toggle
      id={itemKey.toString()}
      size="small"
      value={selected}
      onChange={onSelect}
      className={CSS(Menu.CONTEXT_TARGET, selected && Menu.CONTEXT_SELECTED)}
      textColor={9}
    >
      {GroupIcon != null && <GroupIcon />}
      {group.name}
    </Button.Toggle>
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
          initialValue: "",
          allowEmpty: false,
          label: "Group Name",
        },
        {
          key: "create-group",
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
    <Flex.Box x>
      <Button.Button variant="outlined" size="small" tooltip="Create new symbol group">
        <CreateGroupIcon onClick={handleCreateGroup} />
      </Button.Button>
      <Button.Button
        variant="outlined"
        size="small"
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

const GroupListContextMenu = ({
  keys,
}: Menu.ContextMenuMenuProps): ReactElement | null => {
  const firstKey = keys[0];
  const isRemoteGroup = group.keyZ.safeParse(firstKey).success;
  const item = List.useItem<group.Key, group.Payload>(firstKey);
  const confirmDelete = useConfirmDelete({ type: "Group" });
  const renameModal = Modals.useRename();
  const del = Group.useDelete({
    params: { key: item?.key ?? "" },
    beforeUpdate: async () => {
      if (item == null) return false;
      return await confirmDelete({ name: item.name });
    },
  });
  const rename = Group.useRename({
    params: { key: item?.key ?? "" },
    beforeUpdate: async ({ value }) => {
      if (item == null) return false;
      const newName = await renameModal({ initialValue: value });
      if (newName == null) return false;
      return newName;
    },
  });

  const handleSelect: Menu.MenuProps["onChange"] = {
    delete: () => del.update(),
    rename: () => rename.update(item?.name ?? ""),
  };
  if (!isRemoteGroup) return null;
  return (
    <Menu.Menu level="small" gap="small" onChange={handleSelect}>
      <Menu.Item itemKey="delete">
        <Icon.Delete />
        Delete
      </Menu.Item>
      <Menu.Item itemKey="rename">
        <Icon.Rename />
        Rename
      </Menu.Item>
    </Menu.Menu>
  );
};

const groupListContextMenu = Component.renderProp(GroupListContextMenu);

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
  useEffect(() => fetchMore(), [fetchMore]);
  const menuProps = Menu.useContextMenu();
  return (
    <Select.Frame<group.Key, group.Payload>
      {...data}
      value={value}
      onChange={onChange}
      autoSelectOnNone
    >
      <Menu.ContextMenu {...menuProps} menu={groupListContextMenu}>
        <List.Items onContextMenu={menuProps.open} x gap="small">
          {groupListItem}
        </List.Items>
      </Menu.ContextMenu>
    </Select.Frame>
  );
};

export const Symbols = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const theme = Theming.use();
  const dispatch = useDispatch();
  const groupKey = useSelectSelectedSymbolGroup(layoutKey);
  const setGroupKey = useCallback(
    (group: group.Key) => {
      dispatch(setSelectedSymbolGroup({ key: layoutKey, group }));
    },
    [dispatch, layoutKey],
  );
  const isRemoteGroup = group.keyZ.safeParse(groupKey).success;
  const handleAddElement = useCallback(
    (key: string) => {
      let variant: Schematic.Variant;
      if (isRemoteGroup) variant = "actuator";
      else variant = key as Schematic.Variant;
      const spec = Schematic.SYMBOLS[variant];
      const initialProps = spec.defaultProps(theme);
      if (isRemoteGroup) initialProps.specKey = key;
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
  const g = SchematicSymbol.useGroup.useDirect({ params: {} });
  return (
    <Flex.Box y empty className={CSS.BE("schematic", "symbols")}>
      <Flex.Box x sharp className={CSS.BE("schematic", "symbols", "group", "list")}>
        <Input.Text
          value={search}
          onChange={setSearch}
          placeholder={
            <>
              <Icon.Search />
              Search Symbols
            </>
          }
          size="small"
        />
        {g.data != null && (
          <GroupList
            value={groupKey}
            onChange={setGroupKey}
            symbolGroupID={group.ontologyID(g.data.key)}
          />
        )}
        {g.data != null && (
          <Actions
            symbolGroupID={group.ontologyID(g.data.key)}
            selectedGroup={groupKey}
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
