// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/toolbar/Symbols.css";

import { group, type ontology, schematic } from "@synnaxlabs/client";
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
  Select,
  Status,
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { EmptyAction } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";
import { useSelectSelectedSymbolGroup } from "@/schematic/selectors";
import { setSelectedSymbolGroup } from "@/schematic/slice";
import { createEditLayout } from "@/schematic/symbols/edit/Edit";
import {
  useExport as useExportSymbol,
  useExportGroup,
} from "@/schematic/symbols/export";
import {
  useImport as useImportSymbol,
  useImportGroup,
} from "@/schematic/symbols/import";
import { useAddSymbol } from "@/schematic/symbols/useAddSymbol";
import { useDeleteSymbolGroup } from "@/schematic/symbols/useDeleteSymbolGroup";

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
  const spec = List.useItem<string, Schematic.Symbol.Spec>(itemKey);
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

export interface SymbolListProps {
  onSelect: (key: string) => void;
  groupKey: group.Key;
}

const StaticSymbolList = ({ groupKey, onSelect }: SymbolListProps): ReactElement => {
  const symbols = useMemo(() => {
    const group = Schematic.Symbol.GROUPS.find((g) => g.key === groupKey);
    return Object.values(Schematic.Symbol.REGISTRY).filter((s) =>
      group?.symbols.includes(s.key),
    );
  }, [groupKey]);
  const { data, getItem } = List.useStaticData<string, Schematic.Symbol.Spec>({
    data: symbols,
  });
  return (
    <Select.Frame<string, Schematic.Symbol.Spec>
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

export interface RemoteListItemProps extends List.ItemProps<string> {}

const RemoteListItem = (props: RemoteListItemProps): ReactElement | null => {
  const { itemKey } = props;
  const symbol = List.useItem<string, schematic.symbol.Symbol>(itemKey);
  // Determine if symbol is static or dynamic based on variant or number of states
  const isStatic =
    symbol?.data?.variant === "static" || symbol?.data?.states?.length === 1;
  const variant = isStatic ? "customStatic" : "customActuator";
  const Preview = Schematic.Symbol.REGISTRY[variant].Preview;

  const { startDrag, onDragEnd } = Haul.useDrag({
    type: "Diagram-Elements",
    key: "symbols",
  });

  const handleDragStart = useCallback(() => {
    startDrag([
      { type: "schematic-element", key: variant, data: { specKey: itemKey } },
    ]);
  }, [startDrag, itemKey, variant]);

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
  const exportSymbol = useExportSymbol();
  const rename = Schematic.Symbol.useRename({
    beforeUpdate: async ({ data }) => {
      const { name } = data;
      if (item == null) return false;
      const newName = await renameModal(
        {
          initialValue: name,
          allowEmpty: false,
          label: "Symbol Name",
        },
        {
          name: "Schematic.Symbols.Rename",
          icon: "Schematic",
        },
      );
      if (newName == null) return false;
      return { ...data, name: newName };
    },
  });
  const del = Schematic.Symbol.useDelete({
    beforeUpdate: async () => {
      if (item == null) return false;
      return await confirmDelete({ name: item.name });
    },
  });
  const handleEdit = () => {
    placeLayout(
      createEditLayout({
        args: { key: firstKey, parent: group.ontologyID(props.groupKey) },
      }),
    );
  };
  const handleSelect: Menu.MenuProps["onChange"] = {
    delete: () => del.update(firstKey),
    rename: () => {
      if (item == null) return;
      rename.update(item);
    },
    edit: handleEdit,
    export: () => exportSymbol(firstKey),
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
      <Menu.Item itemKey="export">
        <Icon.Export />
        Export
      </Menu.Item>
    </Menu.Menu>
  );
};

const useCreateSymbol = (selectedGroup: string) => {
  const placeLayout = Layout.usePlacer();
  const handleCreateSymbol = useCallback(() => {
    placeLayout(
      createEditLayout({
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
      message="No symbols found."
      action="Create Symbol"
      onClick={createSymbol}
    />
  );
};

const RemoteSymbolList = ({ groupKey, onSelect }: SymbolListProps): ReactElement => {
  const listData = Schematic.Symbol.useList({
    initialQuery: { parent: group.ontologyID(groupKey) },
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
  const group = List.useItem<group.Key, group.Group & { Icon?: Icon.FC }>(itemKey);
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

const ImportSymbolIcon = Icon.createComposite(Icon.Schematic, {
  bottomRight: Icon.Import,
});

const ImportGroupIcon = Icon.createComposite(Icon.Group, {
  bottomRight: Icon.Import,
});

export interface ActionsProps {
  symbolGroupID?: ontology.ID;
  selectedGroup: string;
}

const Actions = ({
  symbolGroupID,
  selectedGroup,
}: ActionsProps): ReactElement | null => {
  const { updateAsync } = Group.useCreate();
  const rename = Modals.useRename();
  const handleError = Status.useErrorHandler();
  const placeLayout = Layout.usePlacer();
  const importSymbol = useImportSymbol(selectedGroup);
  const importGroup = useImportGroup();

  const handleCreateGroup = useCallback(() => {
    handleError(async () => {
      if (symbolGroupID == null) return;
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
      if (result == null || result.length === 0) return;
      await updateAsync({
        key: uuid.create(),
        name: result,
        parent: symbolGroupID,
      });
    }, "Failed to create group");
  }, [updateAsync, rename, handleError, symbolGroupID]);

  const isRemoteGroup = group.keyZ.safeParse(selectedGroup).success;

  const handleCreateSymbol = useCallback(() => {
    if (!isRemoteGroup || symbolGroupID == null) return;
    placeLayout(
      createEditLayout({
        args: { parent: group.ontologyID(selectedGroup) },
      }),
    );
  }, [isRemoteGroup, placeLayout, selectedGroup]);

  if (symbolGroupID == null) return null;

  return (
    <Flex.Box x>
      <Button.Button
        variant="outlined"
        size="small"
        tooltip="Create new symbol group"
        onClick={handleCreateGroup}
      >
        <CreateGroupIcon />
      </Button.Button>
      <Button.Button
        variant="outlined"
        size="small"
        tooltip="Import symbol group"
        onClick={importGroup}
      >
        <ImportGroupIcon />
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
      <Button.Button
        variant="outlined"
        size="small"
        tooltip="Import symbol"
        disabled={!isRemoteGroup}
        onClick={importSymbol}
      >
        <ImportSymbolIcon />
      </Button.Button>
    </Flex.Box>
  );
};

export interface GroupListProps extends Input.Control<group.Key> {
  symbolGroupID?: ontology.ID;
}

const GroupListContextMenu = ({
  keys,
}: Menu.ContextMenuMenuProps): ReactElement | null => {
  const firstKey = keys[0];
  const isRemoteGroup = group.keyZ.safeParse(firstKey).success;
  const item = List.useItem<group.Key, group.Group>(firstKey);
  const renameModal = Modals.useRename();
  const exportGroup = useExportGroup();
  const deleteSymbolGroup = useDeleteSymbolGroup();
  const rename = Group.useRename({
    beforeUpdate: async ({ data }) => {
      const { name } = data;
      if (item == null) return false;
      const newName = await renameModal(
        { initialValue: name, allowEmpty: false, label: "Group Name" },
        {
          name: "Schematic.Symbols.Rename Group",
          icon: "Group",
        },
      );
      if (newName == null) return false;
      return { ...data, name: newName };
    },
  });

  const handleSelect: Menu.MenuProps["onChange"] = {
    del: () => {
      if (item == null) return;
      deleteSymbolGroup(item);
    },
    rename: () => {
      if (item == null) return;
      rename.update(item);
    },
    export: () => {
      if (item == null) return;
      exportGroup(item);
    },
  };
  if (!isRemoteGroup) return null;
  return (
    <Menu.Menu level="small" gap="small" onChange={handleSelect}>
      <Menu.Item itemKey="del">
        <Icon.Delete />
        Delete
      </Menu.Item>
      <Menu.Item itemKey="rename">
        <Icon.Rename />
        Rename
      </Menu.Item>
      <Menu.Item itemKey="export">
        <Icon.Export />
        Export
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
  const staticData = List.useStaticData<group.Key, group.Group>({
    data: Schematic.Symbol.GROUPS,
  });
  const remoteData = Group.useList({ initialQuery: { parent: symbolGroupID } });
  useEffect(
    () => remoteData.retrieve({ parent: symbolGroupID }),
    [remoteData.retrieve, symbolGroupID],
  );
  const data = List.useCombinedData<group.Key, group.Group>({
    first: staticData,
    second: remoteData,
  });
  const menuProps = Menu.useContextMenu();
  return (
    <Select.Frame<group.Key, group.Group>
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

interface SearchSymbolListProps {
  searchTerm: string;
  onSelect: (key: string) => void;
}

const ALL_STATIC_SYMBOLS = Object.values(Schematic.Symbol.REGISTRY);

const SearchListItem = (props: List.ItemProps<string>): ReactElement | null => {
  const { itemKey } = props;
  const item = List.useItem<string, Schematic.Symbol.Spec | schematic.symbol.Symbol>(
    itemKey,
  );
  if (item == null) return null;
  const isRemote = schematic.symbol.keyZ.safeParse(itemKey).success;
  if (isRemote) return <RemoteListItem {...props} />;
  return <StaticListItem {...props} />;
};

const searchListItem = Component.renderProp(SearchListItem);

const SearchSymbolList = ({
  searchTerm,
  onSelect,
}: SearchSymbolListProps): ReactElement => {
  const remote = Schematic.Symbol.useList({
    initialQuery: { searchTerm },
  });
  const staticData = List.useStaticData<string, Schematic.Symbol.Spec>({
    data: ALL_STATIC_SYMBOLS,
  });
  const { data, getItem, subscribe } = List.useCombinedData<
    string,
    Schematic.Symbol.Spec | schematic.symbol.Symbol
  >({ first: staticData, second: remote });
  const { search } = List.usePager({
    retrieve: useCallback(
      (args) => {
        remote.retrieve(args);
        staticData.retrieve(args);
      },
      [remote.retrieve, staticData.retrieve],
    ),
  });

  useEffect(() => search(searchTerm), [search, searchTerm]);
  return (
    <Select.Frame<string, Schematic.Symbol.Spec | schematic.symbol.Symbol>
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      value={undefined}
      allowNone
      onChange={onSelect}
    >
      <List.Items x className={CSS.BE("schematic", "symbols", "group")} wrap>
        {searchListItem}
      </List.Items>
    </Select.Frame>
  );
};

export const Symbols = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const dispatch = useDispatch();
  const groupKey = useSelectSelectedSymbolGroup(layoutKey);
  const setGroupKey = useCallback(
    (group: group.Key) => dispatch(setSelectedSymbolGroup({ key: layoutKey, group })),
    [dispatch, layoutKey],
  );
  const isRemoteGroup = group.keyZ.safeParse(groupKey).success;
  const addElement = useAddSymbol(dispatch, layoutKey);
  const handleAddElement = useCallback(
    (key: string) => addElement(key, undefined, { specKey: key }),
    [addElement],
  );

  const [searchTerm, setSearchTerm] = useState("");
  const symbolGroup = Schematic.Symbol.useRetrieveGroup({ query: {} });
  const searchMode = searchTerm.length > 0;
  let symbolList = (
    <StaticSymbolList key={groupKey} groupKey={groupKey} onSelect={handleAddElement} />
  );
  if (isRemoteGroup)
    symbolList = (
      <RemoteSymbolList
        key={groupKey}
        groupKey={groupKey}
        onSelect={handleAddElement}
      />
    );
  else if (searchMode)
    symbolList = (
      <SearchSymbolList searchTerm={searchTerm} onSelect={handleAddElement} />
    );
  const symbolGroupID =
    symbolGroup.data?.key != null ? group.ontologyID(symbolGroup.data.key) : undefined;
  return (
    <Flex.Box y empty className={CSS.BE("schematic", "symbols")}>
      <Flex.Box x sharp className={CSS.BE("schematic", "symbols", "group", "list")}>
        <Input.Text
          value={searchTerm}
          onChange={setSearchTerm}
          placeholder={
            <>
              <Icon.Search />
              Search Symbols
            </>
          }
          size="small"
        />
        <GroupList
          value={groupKey}
          onChange={setGroupKey}
          symbolGroupID={symbolGroupID}
        />
        <Actions symbolGroupID={symbolGroupID} selectedGroup={groupKey} />
      </Flex.Box>
      {symbolList}
    </Flex.Box>
  );
};
