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
  List,
  Menu,
  Schematic,
  Select,
  Status,
  Tabs,
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useMemo, useState } from "react";

import { Symbol } from "@/feature/schematic/symbol";
import { useExportGroup } from "@/feature/schematic/symbol/export";
import { useImport as useImportSymbol } from "@/feature/schematic/symbol/import";
import { useImportGroup } from "@/feature/schematic/symbol/ImportGroupModal";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Export } from "@/platform/export";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

const HAUL_DRAG_PROPS: Haul.UseDragProps = {
  type: "Diagram-Elements",
  key: "symbols",
};

const StaticListItem = (props: List.ItemProps<string>): ReactElement | null => {
  const { itemKey } = props;
  const theme = Theming.use();
  const addNode = Schematic.useAddNode();
  const { startDrag, onDragEnd } = Haul.useDrag(HAUL_DRAG_PROPS);
  const variant = itemKey as Schematic.Node.Variant;
  const createParams = useCallback(
    (): Schematic.AddNodeProps => ({ key: id.create(), variant }),
    [variant],
  );
  const handleDragStart = useCallback(
    () => startDrag([Schematic.createHaulItem(createParams())]),
    [startDrag, variant],
  );
  const handleAddNode = useCallback(
    () => addNode(createParams()),
    [addNode, createParams],
  );
  const spec = List.useItem<string, Schematic.Node.Spec>(itemKey);
  const defaultConfig = useMemo(() => spec?.defaultConfig(theme), [spec, theme]);
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
  const symbols = useMemo<Schematic.Node.Spec[]>(() => {
    const g = Schematic.Node.GROUPS.find((g) => g.key === groupKey);
    return Object.values(Schematic.Node.REGISTRY).filter((s) =>
      g?.symbols.includes(s.key),
    ) as unknown as Schematic.Node.Spec[];
  }, [groupKey]);
  const { data, getItem } = List.useStaticData<string, Schematic.Node.Spec>({
    data: symbols,
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
  const { itemKey } = props;
  const symbol = List.useItem<string, schematic.symbol.Symbol>(itemKey);
  const isStatic =
    symbol?.data?.variant === "static" || symbol?.data?.states?.length === 1;
  const variant: Schematic.Node.Variant = isStatic ? "customStatic" : "customActuator";
  const Preview = Schematic.Node.REGISTRY[variant].Preview as React.FC<{
    specKey: string;
    scale?: number;
  }>;
  const addNode = Schematic.useAddNode();
  const { startDrag, onDragEnd } = Haul.useDrag(HAUL_DRAG_PROPS);

  const createParams = useCallback(
    (): Schematic.AddNodeProps => ({ key: id.create(), variant, specKey: itemKey }),
    [variant, itemKey],
  );
  const handleDragStart = useCallback(
    () => startDrag([Schematic.createHaulItem(createParams())]),
    [startDrag, variant, itemKey],
  );
  const handleAddNode = useCallback(
    () => addNode(createParams()),
    [addNode, createParams],
  );
  const { update: rename } = Schematic.Symbol.useRename();
  const handleRename = useCallback(
    (name: string) => rename({ key: itemKey, name }),
    [itemKey, rename],
  );

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
      <Text.MaybeEditable
        id={List.itemNameID(itemKey)}
        level="small"
        value={symbol.name}
        allowDoubleClick={false}
        onChange={handleRename}
      />
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

const RemoteSymbolListContextMenu = ({
  ...rest
}: RemoteSymbolListContextMenuProps): ReactElement => {
  const firstKey = rest.keys[0];
  const item = List.useItem<schematic.symbol.Key, schematic.symbol.Symbol>(firstKey);
  const confirmDelete = Modals.useConfirmDelete({
    type: "Symbol",
    title: "Schematic.Symbols.Delete",
    icon: "Schematic",
  });
  const openEdit = Symbol.Edit.useModal();
  const exportSymbol = Export.use();
  const del = Schematic.Symbol.useDelete({
    beforeUpdate: async () => {
      if (item == null) return false;
      return await confirmDelete({ name: item.name });
    },
  });
  const handleEdit = () => {
    openEdit({ symbolKey: firstKey, parent: group.ontologyID(rest.groupKey) });
  };
  return (
    <ContextMenu.Menu>
      <Menu.Item itemKey="edit" onClick={handleEdit}>
        <Icon.Edit />
        Edit
      </Menu.Item>
      <ContextMenu.RenameItem onClick={() => Text.edit(List.itemNameID(firstKey))} />
      <Menu.Divider />
      <Export.ContextMenuItem
        onClick={() => exportSymbol(schematic.symbol.ontologyID(firstKey))}
      />
      <Menu.Divider />
      <ContextMenu.DeleteItem onClick={() => del.update(firstKey)} />
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const useCreateSymbol = (selectedGroup: string) => {
  const openEdit = Symbol.Edit.useModal();
  const handleCreateSymbol = useCallback(() => {
    openEdit({ parent: group.ontologyID(selectedGroup) });
  }, [openEdit, selectedGroup]);
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
    <Empty.Action
      message="No symbols found."
      action="Create symbol"
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
          emptyContent={
            listData.answered && <RemoteListEmptyContent groupKey={groupKey} />
          }
          wrap
        >
          {remoteListItem}
        </List.Items>
      </Menu.ContextMenu>
    </List.Frame>
  );
};

interface GroupTabProps {
  itemKey: group.Key;
}

const GroupTab = ({ itemKey }: GroupTabProps): ReactElement | null => {
  // Named item rather than group so the client's group namespace stays reachable.
  const item = List.useItem<group.Key, group.Group & { Icon?: Icon.FC }>(itemKey);
  const { update: rename } = Group.useRename();
  const handleRename = useCallback(
    (name: string) => rename({ key: itemKey, name }),
    [itemKey, rename],
  );
  if (item == null) return null;
  const { Icon: GroupIcon } = item;
  // Static groups ship with the Console under non-UUID keys and have no server record.
  const isRemote = group.keyZ.safeParse(itemKey).success;
  return (
    <Tabs.Tab itemKey={itemKey}>
      {GroupIcon != null && <GroupIcon />}
      <Text.MaybeEditable
        id={List.itemNameID(itemKey)}
        level="small"
        value={item.name}
        allowDoubleClick={false}
        onChange={isRemote ? handleRename : undefined}
      />
    </Tabs.Tab>
  );
};

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
  symbolGroupID: ontology.ID;
  selectedGroup: string;
}

const Actions = ({ symbolGroupID, selectedGroup }: ActionsProps): ReactElement => {
  const { updateAsync } = Group.useCreate();
  const rename = Modals.useRename();
  const handleError = Status.useErrorHandler();
  const openEdit = Symbol.Edit.useModal();
  const importSymbol = useImportSymbol(selectedGroup);
  const importGroup = useImportGroup();
  const hasCreateGroupPermission = Access.useCreateGranted(group.TYPE_ONTOLOGY_ID);
  const hasCreateSymbolPermission = Access.useCreateGranted(
    schematic.symbol.TYPE_ONTOLOGY_ID,
  );

  const handleCreateGroup = useCallback(() => {
    handleError(async () => {
      const result = await rename({
        initialValue: "",
        allowEmpty: false,
        title: "Schematic.Symbols.Group.Create",
        icon: <Icon.Group />,
      });
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
    if (!isRemoteGroup) return;
    openEdit({ parent: group.ontologyID(selectedGroup) });
  }, [isRemoteGroup, openEdit, selectedGroup]);

  return (
    <Flex.Box x shrink={0}>
      {hasCreateGroupPermission && (
        <>
          <Button.Button
            variant="outlined"
            size="small"
            tooltip="Create symbol group"
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
            tooltip="Create symbol"
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
  const exportGroup = useExportGroup();
  const deleteSymbolGroup = Symbol.useDeleteGroup();

  if (!isRemoteGroup) return null;
  return (
    <ContextMenu.Menu>
      <ContextMenu.RenameItem onClick={() => Text.edit(List.itemNameID(firstKey))} />
      <Menu.Divider />
      <Export.ContextMenuItem
        onClick={() => {
          if (item != null) exportGroup(item);
        }}
      />
      <Menu.Divider />
      <ContextMenu.DeleteItem
        onClick={() => {
          if (item != null) deleteSymbolGroup(item);
        }}
      />
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
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
  const listProps = List.useCombinedData<group.Key, group.Group>({
    first: staticData,
    second: remoteData,
  });
  const menuProps = Menu.useContextMenu();
  return (
    <Select.Frame<group.Key, group.Group>
      {...listProps}
      value={value}
      onChange={onChange}
      autoSelectOnNone
    >
      <Menu.ContextMenu {...menuProps} menu={groupListContextMenu}>
        <Tabs.Frame x align="center" grow>
          <Tabs.Selector size="small" sizing="content" onContextMenu={menuProps.open}>
            {listProps.data.map((key) => (
              <GroupTab key={key} itemKey={key} />
            ))}
          </Tabs.Selector>
        </Tabs.Frame>
      </Menu.ContextMenu>
    </Select.Frame>
  );
};

interface SearchSymbolListProps {
  searchTerm: string;
}

const SearchListItem = (props: List.ItemProps<string>): ReactElement | null => {
  const { itemKey } = props;
  const item = List.useItem<string, Schematic.Node.Spec | schematic.symbol.Symbol>(
    itemKey,
  );
  if (item == null) return null;
  const isRemote = schematic.symbol.keyZ.safeParse(itemKey).success;
  if (isRemote) return <RemoteListItem {...props} />;
  return <StaticListItem {...props} />;
};

const searchListItem = Component.renderProp(SearchListItem);

const SearchSymbolList = ({ searchTerm }: SearchSymbolListProps): ReactElement => {
  const remote = Schematic.Symbol.useList({
    initialQuery: { searchTerm },
  });
  const staticData = List.useStaticData<string, Schematic.Node.Spec>({
    data: Schematic.Node.STATIC_SPECS,
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
  const dispatch = Session.useDispatch();
  const key = Schematic.useKey();
  const groupKey = Session.Schematic.useSelectSelectedSymbolGroup({ key });
  const setGroupKey = useCallback(
    (group: group.Key) =>
      dispatch(Session.Schematic.setSelectedSymbolGroup({ key, group })),
    [dispatch, key],
  );
  const isRemoteGroup = group.keyZ.safeParse(groupKey).success;

  const [searchTerm, setSearchTerm] = useState("");
  const { data: symbolGroup } = Schematic.Symbol.useResultGroup({});
  const searchMode = searchTerm.length > 0;
  let symbolList = <StaticSymbolList key={groupKey} groupKey={groupKey} />;
  if (isRemoteGroup)
    symbolList = <RemoteSymbolList key={groupKey} groupKey={groupKey} />;
  else if (searchMode) symbolList = <SearchSymbolList searchTerm={searchTerm} />;
  const symbolGroupID = useMemo(
    () => (symbolGroup == null ? undefined : group.ontologyID(symbolGroup.key)),
    [symbolGroup?.key],
  );
  return (
    <Flex.Box y empty className={CSS.BE("schematic", "symbols")}>
      <Flex.Box x sharp className={CSS.BE("schematic", "symbols", "group", "list")}>
        <Input.Text
          value={searchTerm}
          onChange={setSearchTerm}
          placeholder="Search symbols..."
          startContent={<Icon.Search />}
          flush
          className={CSS.BE("schematic", "symbols", "search")}
        />
        <GroupList
          value={groupKey}
          onChange={setGroupKey}
          symbolGroupID={symbolGroupID}
        />
        {symbolGroupID != null && (
          <Actions symbolGroupID={symbolGroupID} selectedGroup={groupKey} />
        )}
      </Flex.Box>
      {symbolList}
    </Flex.Box>
  );
};
