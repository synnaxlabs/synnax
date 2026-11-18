// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, imex, schematic } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Flux, Group, Schematic } from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import {
  createContext,
  type DragEvent,
  type ReactElement,
  use,
  useEffect,
  useState,
} from "react";
import { z } from "zod";

import { Symbol } from "@/feature/schematic/symbol";
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

export interface SymbolsChangeProps {
  /** The symbol that every symbol to change draws, if they all draw the same one. */
  value?: string;
  onChange: <V extends Schematic.Node.Variant>(input: Schematic.Node.Input<V>) => void;
}

const ChangeContext = createContext<SymbolsChangeProps | undefined>(undefined);
ChangeContext.displayName = "Symbols.ChangeContext";

const StaticListItem = (props: List.ItemProps<string>): ReactElement | null => {
  const { itemKey } = props;
  const addNode = Schematic.useAddNode();
  const { startDrag, onDragEnd } = Haul.useDrag(HAUL_DRAG_PROPS);
  const variant = itemKey as Schematic.Node.Variant;
  const createParams = (): Schematic.AddNodeProps => ({
    key: id.create(),
    config: { variant },
  });
  const handleAddNode = () => addNode(createParams());
  const change = use(ChangeContext);
  const spec = List.useItem<string, Schematic.Node.Spec>(itemKey);
  const config = spec == null ? null : Schematic.Node.createConfig({ variant });
  if (spec == null || config == null) return null;
  const { name, Preview } = spec;
  const handleDragStart = (event: DragEvent<HTMLElement>) =>
    startDrag({
      event,
      items: [Schematic.createHaulItem(createParams())],
      preview: <Preview {...config} scale={1} />,
      anchor: "corner",
    });
  return (
    <List.Item
      className={CSS.cls(CSS.BE("schematic-symbols", "button"))}
      align="center"
      gap="tiny"
      draggable={change == null}
      selected={change?.value === variant}
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
      onClick={change == null ? handleAddNode : () => change.onChange({ variant })}
      y
      {...props}
    >
      <Text.Text level="small">{name}</Text.Text>
      <Flex.Box align="center" justify="center" grow>
        <Preview {...config} scale={0.75} />
      </Flex.Box>
    </List.Item>
  );
};

const staticListItem = Component.renderProp(StaticListItem);

export interface SymbolListProps {
  groupKey: group.Key;
}

const StaticSymbolList = ({ groupKey }: SymbolListProps): ReactElement => {
  const g = Schematic.Node.GROUPS.find((g) => g.key === groupKey);
  const symbols = Schematic.Node.STATIC_SPECS.filter((s) => g?.symbols.includes(s.key));
  const { data, getItem } = List.useStaticData<string, Schematic.Node.Spec>({
    data: symbols,
  });
  return (
    <List.Frame<string, Schematic.Node.Spec> data={data} getItem={getItem}>
      <List.Scroll x className={CSS.BE("schematic", "symbols", "group")} wrap>
        <List.Items>{staticListItem}</List.Items>
      </List.Scroll>
    </List.Frame>
  );
};

export interface RemoteListItemProps extends List.ItemProps<string> {}

const RemoteListItem = (props: RemoteListItemProps): ReactElement | null => {
  const { itemKey } = props;
  const symbol = List.useItem<string, schematic.symbol.Symbol>(itemKey);
  const variant: Schematic.Node.CustomVariant =
    symbol == null ? "custom_actuator" : Schematic.Node.customVariant(symbol);
  const Preview = Schematic.Node.REGISTRY[variant].Preview as React.FC<{
    specKey: string;
    scale?: number;
  }>;
  const addNode = Schematic.useAddNode();
  const { startDrag, onDragEnd } = Haul.useDrag(HAUL_DRAG_PROPS);

  const createParams = (): Schematic.AddNodeProps<Schematic.Node.CustomVariant> => ({
    key: id.create(),
    config: { variant, specKey: itemKey },
  });
  const handleAddNode = () => addNode(createParams());
  const change = use(ChangeContext);
  const { update: rename } = Schematic.Symbol.useRename();
  const canRename = Access.useUpdateGranted(schematic.symbol.resourceID(itemKey));
  const handleRename = (name: string) => rename({ key: itemKey, name });

  if (symbol == null) return null;
  // The node's own drawing reads the symbol through a hook, which a preview cannot.
  const handleDragStart = (event: DragEvent<HTMLElement>) =>
    startDrag({
      event,
      items: [Schematic.createHaulItem(createParams())],
      preview: (
        <Schematic.Node.Primitive.Div orientation="left">
          <Schematic.Node.Custom.Drawing spec={symbol.data} />
        </Schematic.Node.Primitive.Div>
      ),
      anchor: "corner",
    });

  return (
    <Select.Item
      className={CSS.cls(CSS.BE("schematic-symbols", "button"))}
      align="center"
      gap="tiny"
      draggable={change == null}
      selected={change?.value === itemKey}
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
      onClick={
        change == null
          ? handleAddNode
          : () => change.onChange({ variant, specKey: itemKey })
      }
      y
      {...props}
    >
      <Text.MaybeEditable
        id={List.itemNameID(itemKey)}
        level="small"
        value={symbol.name}
        allowDoubleClick={false}
        onChange={canRename ? handleRename : undefined}
      />
      <Flex.Box align="center" justify="center" grow>
        <Preview specKey={itemKey} scale={0.75} />
      </Flex.Box>
    </Select.Item>
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
  const canRename = Access.useUpdateGranted(schematic.symbol.resourceID(firstKey));
  const canDelete = Access.useDeleteGranted(schematic.symbol.resourceID(firstKey));
  const confirmDelete = Modals.useConfirmDelete({
    type: "Symbol",
    title: "Schematic.Symbol.Delete",
    icon: "Schematic",
  });
  const openEdit = Symbol.Edit.useModal();
  const exportSymbol = Export.useResource();
  const del = Schematic.Symbol.useDelete({
    beforeUpdate: async () => {
      if (item == null) return false;
      return await confirmDelete({ name: item.name });
    },
  });
  const handleEdit = () => {
    openEdit({ symbolKey: firstKey });
  };
  return (
    <ContextMenu.Menu>
      <Menu.Item itemKey="edit" onClick={handleEdit}>
        <Icon.Edit />
        Edit
      </Menu.Item>
      {canRename && (
        <ContextMenu.RenameItem onClick={() => Text.edit(List.itemNameID(firstKey))} />
      )}
      <Menu.Divider />
      <Export.ContextMenuItem
        onClick={() => exportSymbol(schematic.symbol.resourceID(firstKey))}
      />
      <Menu.Divider />
      {canDelete && <ContextMenu.DeleteItem onClick={() => del.update(firstKey)} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const useCreateSymbol = (selectedGroup: string) => {
  const openEdit = Symbol.Edit.useModal();
  const handleCreateSymbol = () => {
    openEdit({ group: selectedGroup });
  };
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
      message="No symbols found"
      action="Create symbol"
      onClick={createSymbol}
    />
  );
};

const RemoteSymbolList = ({ groupKey }: SymbolListProps): ReactElement => {
  const { data, getItem, subscribe, more, answered } = Schematic.Symbol.useList({
    query: { groups: [groupKey] },
  });
  const menuProps = Menu.useContextMenu();
  return (
    <List.Frame<string, schematic.symbol.Symbol>
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={more}
    >
      <Menu.ContextMenu
        {...menuProps}
        menu={(props) => <RemoteSymbolListContextMenu {...props} groupKey={groupKey} />}
      >
        <List.Scroll
          x
          className={CSS.BE("schematic", "symbols", "group")}
          onContextMenu={menuProps.open}
          wrap
        >
          <List.Items
            emptyContent={answered && <RemoteListEmptyContent groupKey={groupKey} />}
          >
            {remoteListItem}
          </List.Items>
        </List.Scroll>
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
  const canRename = Access.useUpdateGranted(group.resourceID(itemKey));
  const handleRename = (name: string) => rename({ key: itemKey, name });
  if (item == null) return null;
  const { Icon: GroupIcon } = item;
  // Static groups ship with the Console under non-UUID keys and have no server record.
  const isRemote = z.validate(group.keyZ, itemKey);
  return (
    <Tabs.Tab itemKey={itemKey}>
      {GroupIcon != null && <GroupIcon />}
      <Text.MaybeEditable
        id={List.itemNameID(itemKey)}
        level="small"
        value={item.name}
        allowDoubleClick={false}
        overflow="fade"
        onChange={isRemote && canRename ? handleRename : undefined}
      />
    </Tabs.Tab>
  );
};

const CreateGroupIcon = Icon.createFromPath(
  "group-create",
  "M21.38,7.74v11.19c0,0.41 -0.34,0.75 -0.75,0.75h-17.25c-0.41,0 -0.75,-0.34 -0.75,-0.75v-13.87c0,-0.41 0.34,-0.75 0.75,-0.75h5.96c0.05,0 0.09,0.02 0.13,0.05l2.75,2.63h8.41c0.41,0 0.75,0.34 0.75,0.75zM8,12.25v2.5h2.75v2.75h2.5v-2.75h2.75v-2.5h-2.75v-2.75h-2.5v2.75z",
);

const ImportSymbolIcon = Icon.createFromPath(
  "schematic-import",
  "M14.38,16.38h-12.82c-0.24,0 -0.46,-0.13 -0.58,-0.33c-0.12,-0.21 -0.12,-0.46 -0.01,-0.67l6.41,-11.47c0.12,-0.21 0.34,-0.35 0.59,-0.35c0.24,0 0.47,0.13 0.59,0.35l2.64,4.72c0.04,0.07 0.06,0.16 0.06,0.24l0,2.37c0,0.01 0,0.02 0,0.03c0,0.41 0.17,0.78 0.44,1.05c0,0 0,0 0,0c0.16,0.16 0.36,0.29 0.59,0.37c0.15,0.05 0.3,0.07 0.47,0.07h0.46c0.18,0 0.35,0.1 0.44,0.26l1.32,2.36c0.12,0.21 0.11,0.46 -0.01,0.67c-0.12,0.21 -0.34,0.33 -0.58,0.33zM20.57,13.01c0.27,0.62 0.45,1.29 0.52,2c0.35,3.44 -1.96,6.6 -5.35,7.29c-3.39,0.7 -6.76,-1.28 -7.8,-4.59h6.44c0.72,0 1.38,-0.38 1.75,-1c0.36,-0.62 0.37,-1.39 0.02,-2.01l-0.68,-1.22c-0.09,-0.15 -0.08,-0.34 0.01,-0.5c0.09,-0.15 0.25,-0.25 0.43,-0.25l4.23,0c0.19,0 0.35,0.1 0.44,0.26zM17.5,2l4.5,4.95h-3.25v4.05h-2.5v-4.05h-3.25z",
);

const ImportGroupIcon = Icon.createFromPath(
  "group-import",
  "M21.38,7.74v11.19c0,0.41 -0.34,0.75 -0.75,0.75h-17.25c-0.41,0 -0.75,-0.34 -0.75,-0.75v-13.87c0,-0.41 0.34,-0.75 0.75,-0.75h5.96c0.05,0 0.09,0.02 0.13,0.05l2.75,2.63h8.41c0.41,0 0.75,0.34 0.75,0.75zM16,13.9l-4,-4.4l-4,4.4h2.75v3.6h2.5v-3.6z",
);

export interface ActionsProps {
  selectedGroup: string;
  onSelectGroup: (group: group.Key) => void;
}

const Actions = ({ selectedGroup, onSelectGroup }: ActionsProps): ReactElement => {
  const { updateAsync } = Group.useCreate();
  const rename = Modals.useRename();
  const handleError = Status.useErrorHandler();
  const openEdit = Symbol.Edit.useModal();
  const importSymbol = Symbol.useImport();
  const importGroup = Symbol.useImportGroup();
  const hasCreateGroupPermission = Access.useCreateGranted(group.TYPE_RESOURCE_ID);
  const hasCreateSymbolPermission = Access.useCreateGranted(
    schematic.symbol.TYPE_RESOURCE_ID,
  );

  const handleCreateGroup = () => {
    handleError(async () => {
      const result = await rename({
        initialValue: "",
        allowEmpty: false,
        title: "Schematic.Symbol.Group.Create",
        icon: <Icon.Group />,
      });
      if (result == null || result.length === 0) return;
      const key = uuid.create();
      const created = await updateAsync({
        key,
        name: result,
        scope: schematic.symbol.TYPE_RESOURCE_ID.type,
      });
      if (created) onSelectGroup(key);
    }, "Failed to create group");
  };

  const isRemoteGroup = z.validate(group.keyZ, selectedGroup);

  const handleCreateSymbol = () => {
    if (!isRemoteGroup) return;
    openEdit({ group: selectedGroup });
  };

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
            <Schematic.CreateIcon />
          </Button.Button>
          <Button.Button
            variant="outlined"
            size="small"
            tooltip="Import symbol"
            disabled={!isRemoteGroup}
            onClick={() => importSymbol(selectedGroup)}
          >
            <ImportSymbolIcon />
          </Button.Button>
        </>
      )}
    </Flex.Box>
  );
};

export interface GroupListProps extends Input.Control<group.Key> {}

const GroupListContextMenu = ({
  keys,
}: Menu.ContextMenuMenuProps): ReactElement | null => {
  const firstKey = keys[0];
  const isRemoteGroup = z.validate(group.keyZ, firstKey);
  const item = List.useItem<group.Key, group.Group>(firstKey);
  const canRename = Access.useUpdateGranted(group.resourceID(firstKey));
  const canDelete = Access.useDeleteGranted(group.resourceID(firstKey));
  const exportGroup = Export.use();
  const deleteSymbolGroup = Symbol.useDeleteGroup();

  if (!isRemoteGroup) return null;
  return (
    <ContextMenu.Menu>
      {canRename && (
        <ContextMenu.RenameItem onClick={() => Text.edit(List.itemNameID(firstKey))} />
      )}
      <Menu.Divider />
      <Export.ContextMenuItem
        onClick={() => {
          if (item != null)
            exportGroup({
              stream: (client) =>
                client.schematics.symbols.exportGroup(item.key, imex.JSON_OPTIONS),
              name: item.name,
              extension: "zip",
            });
        }}
      />
      <Menu.Divider />
      {canDelete && (
        <ContextMenu.DeleteItem
          onClick={() => {
            if (item != null) deleteSymbolGroup(item);
          }}
        />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const groupListContextMenu = Component.renderProp(GroupListContextMenu);

const GroupList = ({ value, onChange }: GroupListProps): ReactElement => {
  const staticData = List.useStaticData<group.Key, group.Group>({
    data: Schematic.Node.GROUPS,
  });
  // A tab strip cannot page, so it reads every group at once.
  const remoteData = Flux.useList((client) => client.groups, {
    name: "symbol groups",
    query: Schematic.Symbol.GROUP_QUERY,
    pageSize: 1000,
  });
  useEffect(() => remoteData.more(), []);
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
  const isRemote = z.validate(schematic.symbol.keyZ, itemKey);
  if (isRemote) return <RemoteListItem {...props} />;
  return <StaticListItem {...props} />;
};

const searchListItem = Component.renderProp(SearchListItem);

const SearchSymbolList = ({ searchTerm }: SearchSymbolListProps): ReactElement => {
  const remote = Schematic.Symbol.useList();
  const staticData = List.useStaticData<string, Schematic.Node.Spec>({
    data: Schematic.Node.STATIC_SPECS,
  });
  const { data, getItem, subscribe } = List.useCombinedData<
    string,
    Schematic.Node.Spec | schematic.symbol.Symbol
  >({ first: staticData, second: remote });
  const { search: searchRemote } = remote;
  const { search: searchStatic } = staticData;
  useEffect(() => {
    searchRemote(searchTerm);
    searchStatic(searchTerm);
  }, [searchRemote, searchStatic, searchTerm]);
  return (
    <List.Frame<string, Schematic.Node.Spec | schematic.symbol.Symbol>
      data={data}
      getItem={getItem}
      subscribe={subscribe}
    >
      <List.Scroll x className={CSS.BE("schematic", "symbols", "group")} wrap>
        <List.Items>{searchListItem}</List.Items>
      </List.Scroll>
    </List.Frame>
  );
};

export interface SymbolsProps {
  /** Makes a click on a symbol change the given symbols in place of adding one. */
  change?: SymbolsChangeProps;
}

export const Symbols = ({ change }: SymbolsProps): ReactElement => {
  const dispatch = Session.useDispatch();
  const key = Schematic.useKey();
  const groupKey = Session.Schematic.useSelectSelectedSymbolGroup({ key });
  const setGroupKey = (group: group.Key) =>
    dispatch(Session.Schematic.setSelectedSymbolGroup({ key, group }));
  const isRemoteGroup = z.validate(group.keyZ, groupKey);

  const [searchTerm, setSearchTerm] = useState("");
  const searchMode = searchTerm.length > 0;
  let symbolList = <StaticSymbolList key={groupKey} groupKey={groupKey} />;
  if (isRemoteGroup)
    symbolList = <RemoteSymbolList key={groupKey} groupKey={groupKey} />;
  else if (searchMode) symbolList = <SearchSymbolList searchTerm={searchTerm} />;
  return (
    <ChangeContext value={change}>
      <Flex.Box
        y
        empty
        className={CSS.cls(
          CSS.BE("schematic", "symbols"),
          change != null && CSS.M("changing"),
        )}
      >
        <Flex.Box x sharp className={CSS.BE("schematic", "symbols", "group", "list")}>
          <Input.Text
            value={searchTerm}
            onChange={setSearchTerm}
            placeholder="Search symbols..."
            startContent={<Icon.Search />}
            flush
            className={CSS.BE("schematic", "symbols", "search")}
          />
          <GroupList value={groupKey} onChange={setGroupKey} />
          <Actions selectedGroup={groupKey} onSelectGroup={setGroupKey} />
        </Flex.Box>
        {symbolList}
      </Flex.Box>
    </ChangeContext>
  );
};
