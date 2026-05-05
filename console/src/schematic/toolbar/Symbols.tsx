// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, type ontology, schematic } from "@synnaxlabs/client";
import {
  Access,
  Button,
  Component,
  Flex,
  Group,
  Haul,
  Icon,
  Input,
  Key,
  List,
  Menu,
  Schematic,
  Select,
  Status,
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu, EmptyAction } from "@/components";
import { CSS } from "@/css";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";
import { createSymbolHaulItem } from "@/schematic/haul";
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
import { type AddNodeProps, useAddNode } from "@/schematic/symbols/useAddNode";
import { useDeleteSymbolGroup } from "@/schematic/symbols/useDeleteSymbolGroup";

const HAUL_TYPE = "schematic_symbol";
const USE_DRAG_PROPS: Haul.UseDragProps = { type: HAUL_TYPE, key: "symbols" };

const StaticListItem = (
  props: List.ItemProps<Schematic.Node.Variant>,
): ReactElement | null => {
  const { itemKey: variant } = props;
  const theme = Theming.use();
  const layoutKey = Key.use<string>("Schematic.Toolbar.StaticListItem");
  const { startDrag, onDragEnd } = Haul.useDrag(USE_DRAG_PROPS);
  const createAddParams = useCallback(() => ({ key: id.create(), variant }), [variant]);
  const handleDragStart = useCallback(
    () => startDrag([createSymbolHaulItem(createAddParams())]),
    [startDrag, createAddParams],
  );
  const spec = List.useItem<string, Schematic.Node.Spec>(variant);
  const defaultConfig = useMemo(() => spec?.defaultConfig(theme), [spec, theme]);
  const addNode = useAddNode(layoutKey);
  const handleAddNode = useCallback(
    () => addNode(createAddParams()),
    [createAddParams],
  );
  if (spec == null || defaultConfig == null) return null;
  const { name, Preview } = spec;
  return (
    <List.Item
      className={CSS(CSS.BE("schematic-symbols", "button"))}
      align="center"
      gap="tiny"
      draggable
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
      onClick={handleAddNode}
      y
      {...props}
    >
      <Text.Text level="small">{name}</Text.Text>
      <Flex.Box align="center" justify="center" grow>
        <Preview {...defaultConfig} scale={0.75} />
      </Flex.Box>
    </List.Item>
  );
};

const staticListItem = Component.renderProp(StaticListItem);

export interface SymbolListProps {
  groupKey: group.Key;
}

const StaticSymbolList = ({ groupKey }: SymbolListProps): ReactElement => {
  const symbols = useMemo(() => {
    const group = Schematic.Node.GROUPS.find((g) => g.key === groupKey);
    if (group == null) return [];
    return group?.symbols.map((k) => Schematic.Node.REGISTRY[k]);
  }, [groupKey]);
  const { data, getItem } = List.useStaticData<string, Schematic.Node.Spec>({
    data: symbols as Schematic.Node.Spec[],
  });
  return (
    <List.Frame<string, Schematic.Node.Spec> data={data} getItem={getItem}>
      <List.Items x className={CSS.BE("schematic", "symbols", "group")} wrap>
        {staticListItem}
      </List.Items>
    </List.Frame>
  );
};

export interface RemoteListItemProps extends List.ItemProps<string> {}

const RemoteListItem = (props: RemoteListItemProps): ReactElement | null => {
  const { itemKey: specKey } = props;
  const symbol = List.useItem<string, schematic.symbol.Symbol>(specKey);
  const addNodeProps = useMemo((): AddNodeProps => {
    const isStatic =
      symbol?.data?.variant === "static" || symbol?.data?.states?.length === 1;
    const variant = isStatic ? "customStatic" : "customActuator";
    return { key: id.create(), variant, specKey };
  }, [symbol?.data.variant, symbol?.data.states, specKey]);

  const Preview = Schematic.Node.REGISTRY[addNodeProps.variant].Preview as React.FC<{
    specKey: string;
    scale?: number;
  }>;
  const layoutKey = Key.use<string>("Schematic.Toolbar.RemoteListItem");
  const addNode = useAddNode(layoutKey);

  const { startDrag, onDragEnd } = Haul.useDrag(USE_DRAG_PROPS);

  const handleDragStart = useCallback(
    () => startDrag([createSymbolHaulItem(addNodeProps)]),
    [startDrag, addNodeProps],
  );
  const handleAddNode = useCallback(() => addNode(addNodeProps), [addNodeProps]);

  if (symbol == null) return null;

  return (
    <Select.ListItem
      className={CSS(CSS.BE("schematic-symbols", "button"))}
      align="center"
      gap="tiny"
      draggable
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
      onClick={handleAddNode}
      y
      {...props}
    >
      <Text.Text level="small">{symbol.name}</Text.Text>
      <Flex.Box align="center" justify="center" grow>
        <Preview specKey={specKey} scale={0.75} />
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
    type: "Schematic.Node",
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
          name: "Schematic.Symbol.Rename",
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
  return (
    <ContextMenu.Menu>
      <ContextMenu.DeleteItem onClick={() => del.update(firstKey)} />
      <ContextMenu.RenameItem
        onClick={() => {
          if (item != null) rename.update(item);
        }}
      />
      <Menu.Item itemKey="edit" onClick={handleEdit}>
        <Icon.Edit />
        Edit
      </Menu.Item>
      <Export.ContextMenuItem onClick={() => exportSymbol(firstKey)} />
    </ContextMenu.Menu>
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

const RemoteSymbolList = ({ groupKey }: SymbolListProps): ReactElement => {
  const listData = Schematic.Symbol.useList({
    initialQuery: { parent: group.ontologyID(groupKey) },
  });
  const { fetchMore } = List.usePager({ retrieve: listData.retrieve });
  useEffect(() => fetchMore(), [fetchMore]);
  const menuProps = Menu.useContextMenu();
  return (
    <List.Frame<string, schematic.symbol.Symbol> {...listData}>
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
    </List.Frame>
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
  const hasCreateGroupPermission = Access.useCreateGranted(group.TYPE_ONTOLOGY_ID);
  const hasCreateSymbolPermission = Access.useCreateGranted(
    schematic.symbol.TYPE_ONTOLOGY_ID,
  );

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
          name: "Schematic.Nodes.Create Group",
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
      {hasCreateGroupPermission && (
        <>
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
        </>
      )}
      {hasCreateSymbolPermission && (
        <>
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
        </>
      )}
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

  if (!isRemoteGroup) return null;
  return (
    <ContextMenu.Menu>
      <ContextMenu.DeleteItem
        onClick={() => {
          if (item != null) deleteSymbolGroup(item);
        }}
      />
      <ContextMenu.RenameItem
        onClick={() => {
          if (item != null) rename.update(item);
        }}
      />
      <Export.ContextMenuItem
        onClick={() => {
          if (item != null) exportGroup(item);
        }}
      />
    </ContextMenu.Menu>
  );
};

const groupListContextMenu = Component.renderProp(GroupListContextMenu);

const GroupList = ({
  value,
  onChange,
  symbolGroupID,
}: GroupListProps): ReactElement => {
  const staticData = List.useStaticData<group.Key, group.Group>({
    data: Schematic.Node.GROUPS,
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
}

export const CUSTOM_VARIANTS = new Set(["customActuator", "customStatic"]);
export const ALL_STATIC_SYMBOLS = Object.values(Schematic.Node.REGISTRY).filter(
  (s) => !CUSTOM_VARIANTS.has(s.key),
);

const SearchListItem = (props: List.ItemProps<string>): ReactElement | null => {
  const { itemKey } = props;
  const item = List.useItem<string, Schematic.Node.Spec | schematic.symbol.Symbol>(
    itemKey,
  );
  if (item == null) return null;
  const isRemote = schematic.symbol.keyZ.safeParse(itemKey).success;
  if (isRemote) return <RemoteListItem {...props} />;
  const staticItemKey = itemKey as Schematic.Node.Variant;
  return <StaticListItem {...props} itemKey={staticItemKey} key={staticItemKey} />;
};

const searchListItem = Component.renderProp(SearchListItem);

const SearchSymbolList = ({ searchTerm }: SearchSymbolListProps): ReactElement => {
  const remote = Schematic.Symbol.useList({ initialQuery: { searchTerm } });
  const staticData = List.useStaticData<
    string,
    Schematic.Node.Spec<Schematic.Node.Variant, object>
  >({
    data: ALL_STATIC_SYMBOLS as Schematic.Node.Spec<Schematic.Node.Variant, object>[],
  });
  const { data, getItem, subscribe } = List.useCombinedData<
    string,
    Schematic.Node.Spec | schematic.symbol.Symbol
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
    <List.Frame<string, Schematic.Node.Spec | schematic.symbol.Symbol>
      data={data}
      getItem={getItem}
      subscribe={subscribe}
    >
      <List.Items x className={CSS.BE("schematic", "symbols", "group")} wrap>
        {searchListItem}
      </List.Items>
    </List.Frame>
  );
};

export const Symbols = (): ReactElement => {
  const dispatch = useDispatch();
  const layoutKey = Key.use<string>("Schematic.Nodes");
  const groupKey = useSelectSelectedSymbolGroup(layoutKey);
  const setGroupKey = useCallback(
    (group: group.Key) => dispatch(setSelectedSymbolGroup({ key: layoutKey, group })),
    [dispatch, layoutKey],
  );
  const isRemoteGroup = group.keyZ.safeParse(groupKey).success;
  const [searchTerm, setSearchTerm] = useState("");
  const symbolGroup = Schematic.Symbol.useRetrieveGroup({ query: {} });
  const searchMode = searchTerm.length > 0;
  let symbolList = <StaticSymbolList key={groupKey} groupKey={groupKey} />;
  if (isRemoteGroup)
    symbolList = <RemoteSymbolList key={groupKey} groupKey={groupKey} />;
  else if (searchMode) symbolList = <SearchSymbolList searchTerm={searchTerm} />;
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
