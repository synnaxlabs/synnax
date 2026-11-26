// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, group, imex, schematic } from "@synnaxlabs/client";
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
import { Access, Flux, Group, Schematic, Synnax } from "@synnaxlabs/pluto";
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
import { FLAGS } from "@/flags";
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
  /** Takes the picked symbol, with its spec when it draws as a symbol node. */
  onChange: <V extends Schematic.Node.Variant>(
    input: Schematic.Node.Input<V>,
    spec?: schematic.symbol.Spec,
  ) => void;
}

const ChangeContext = createContext<SymbolsChangeProps | undefined>(undefined);
ChangeContext.displayName = "Symbols.ChangeContext";

const createSymbolParams = (
  ref: schematic.symbol.Ref,
  label: string,
): Schematic.AddNodeProps<"symbol"> => ({
  key: id.create(),
  config: { variant: "symbol", spec: ref, label: { label } },
});

const StaticListItem = (props: List.ItemProps<string>): ReactElement | null => {
  const { itemKey } = props;
  const addNode = Schematic.useAddNode();
  const { startDrag, onDragEnd } = Haul.useDrag(HAUL_DRAG_PROPS);
  const variant = itemKey as Schematic.Node.Variant;
  const spec = List.useItem<string, Schematic.Node.Spec>(itemKey);
  const change = use(ChangeContext);
  const config = spec == null ? null : Schematic.Node.createConfig({ variant });
  if (spec == null || config == null) return null;
  const { name, Preview } = spec;
  const builtIn = FLAGS.symbolNodes ? Schematic.Node.BUILT_INS.get(variant) : undefined;
  const createParams = (): Schematic.AddNodeProps<"symbol"> | Schematic.AddNodeProps =>
    builtIn != null
      ? createSymbolParams({ type: "builtin", name: variant }, spec.label ?? name)
      : { key: id.create(), config: { variant } };
  const handleAddNode = () => addNode(createParams());
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
      onClick={
        change == null
          ? handleAddNode
          : () => change.onChange(createParams().config, builtIn)
      }
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

/** useCreateGroup asks for a name and creates a custom symbol group with it. */
const useCreateGroup = (): (() => Promise<group.Key | undefined>) => {
  const { updateAsync } = Group.useCreate();
  const rename = Modals.useRename();
  return async () => {
    const name = await rename({
      initialValue: "",
      allowEmpty: false,
      title: "Schematic.Symbol.Group.Create",
      icon: <Icon.Group />,
    });
    if (name == null || name.length === 0) return undefined;
    const key = uuid.create();
    const created = await updateAsync({
      key,
      name,
      scope: schematic.symbol.TYPE_RESOURCE_ID.type,
    });
    return created ? key : undefined;
  };
};

interface DuplicateToProps {
  name: string;
  spec: schematic.symbol.Spec;
  onSelectGroup: (group: group.Key) => void;
}

/** DuplicateTo copies a built-in symbol into a custom group and opens the copy. */
const DuplicateTo = ({
  name,
  spec,
  onSelectGroup,
}: DuplicateToProps): ReactElement | null => {
  const client = Synnax.use();
  const groups = Group.useResultList(Schematic.Symbol.GROUP_QUERY);
  const canCreateGroup = Access.useCreateGranted(group.TYPE_RESOURCE_ID);
  const createGroup = useCreateGroup();
  const openEdit = Symbol.Edit.useModal();
  const handleError = Status.useErrorHandler();
  const duplicate = async (key: group.Key | undefined) => {
    if (key == null) return;
    if (client == null) throw new DisconnectedError();
    const copy = await client.schematics.symbols.create({
      name: `${name} (copy)`,
      group: key,
      data: spec,
    });
    onSelectGroup(key);
    openEdit({ symbolKey: copy.key });
  };
  const list = groups.variant === "success" ? groups.data : [];
  const handleDuplicate = (key?: group.Key) =>
    handleError(
      async () => await duplicate(key ?? (await createGroup())),
      "Failed to duplicate symbol",
    );
  if (list.length === 0 && !canCreateGroup) return null;
  return (
    <Menu.Sub itemKey="duplicateTo" textValue="Duplicate to">
      <Menu.SubTrigger>
        <Icon.Copy />
        Duplicate to
      </Menu.SubTrigger>
      <Menu.Dialog aria-label="Duplicate to">
        <Menu.Menu gap="small">
          {list.map((g) => (
            <Menu.Item
              key={g.key}
              itemKey={g.key}
              onClick={() => handleDuplicate(g.key)}
            >
              <Icon.Group />
              {g.name}
            </Menu.Item>
          ))}
          {canCreateGroup && (
            <Menu.Item itemKey="newGroup" onClick={() => handleDuplicate()}>
              <Icon.Add />
              New group…
            </Menu.Item>
          )}
        </Menu.Menu>
      </Menu.Dialog>
    </Menu.Sub>
  );
};

interface StaticSymbolListContextMenuProps extends Menu.ContextMenuMenuProps {
  onSelectGroup: (group: group.Key) => void;
}

const StaticSymbolListContextMenu = ({
  keys,
  onSelectGroup,
}: StaticSymbolListContextMenuProps): ReactElement => {
  const variant = keys[0];
  const item = List.useItem<string, Schematic.Node.Spec>(variant);
  const spec = Schematic.Node.BUILT_INS.get(variant);
  const canCreate = Access.useCreateGranted(schematic.symbol.TYPE_RESOURCE_ID);
  return (
    <ContextMenu.Menu>
      {canCreate && item != null && spec != null && (
        <DuplicateTo name={item.name} spec={spec} onSelectGroup={onSelectGroup} />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

interface StaticSymbolListProps extends SymbolListProps {
  onSelectGroup: (group: group.Key) => void;
}

const StaticSymbolList = ({
  groupKey,
  onSelectGroup,
}: StaticSymbolListProps): ReactElement => {
  const g = Schematic.Node.GROUPS.find((g) => g.key === groupKey);
  const symbols = Schematic.Node.STATIC_SPECS.filter((s) => g?.symbols.includes(s.key));
  const { data, getItem } = List.useStaticData<string, Schematic.Node.Spec>({
    data: symbols,
  });
  const menuProps = Menu.useContextMenu();
  return (
    <List.Frame<string, Schematic.Node.Spec> data={data} getItem={getItem}>
      <Menu.ContextMenu
        {...menuProps}
        menu={(props) => (
          <StaticSymbolListContextMenu {...props} onSelectGroup={onSelectGroup} />
        )}
      >
        <List.Scroll
          x
          className={CSS.BE("schematic", "symbols", "group")}
          onContextMenu={menuProps.open}
          wrap
        >
          <List.Items>{staticListItem}</List.Items>
        </List.Scroll>
      </Menu.ContextMenu>
    </List.Frame>
  );
};

export interface RemoteListItemProps extends List.ItemProps<string> {}

const RemoteListItem = (props: RemoteListItemProps): ReactElement | null => {
  const { itemKey } = props;
  const symbol = List.useItem<string, schematic.symbol.Symbol>(itemKey);
  const paints = Schematic.Node.Custom.usePaints(symbol?.data, 0);
  const variant: Schematic.Node.CustomVariant =
    symbol == null ? "custom_actuator" : Schematic.Node.customVariant(symbol);
  const Preview = Schematic.Node.REGISTRY[variant].Preview as React.FC<{
    specKey: string;
    scale?: number;
  }>;
  const addNode = Schematic.useAddNode();
  const { startDrag, onDragEnd } = Haul.useDrag(HAUL_DRAG_PROPS);

  const change = use(ChangeContext);
  const { update: rename } = Schematic.Symbol.useRename();
  const canRename = Access.useUpdateGranted(schematic.symbol.resourceID(itemKey));
  const handleRename = (name: string) => rename({ key: itemKey, name });

  if (symbol == null) return null;
  const createParams = ():
    | Schematic.AddNodeProps<"symbol">
    | Schematic.AddNodeProps<Schematic.Node.CustomVariant> =>
    FLAGS.symbolNodes
      ? createSymbolParams({ type: "custom", key: itemKey }, symbol.name)
      : { key: id.create(), config: { variant, specKey: itemKey } };
  const handleAddNode = () => addNode(createParams());
  // The node's own drawing reads the symbol through a hook, which a preview cannot.
  const handleDragStart = (event: DragEvent<HTMLElement>) =>
    startDrag({
      event,
      items: [Schematic.createHaulItem(createParams())],
      preview: (
        <Schematic.Node.Primitive.Div orientation="left">
          <Schematic.Node.Custom.Drawing spec={symbol.data} paints={paints} />
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
          : () => change.onChange(createParams().config, symbol.data)
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

const RemoteSymbolListContextMenu = ({
  ...rest
}: Menu.ContextMenuMenuProps): ReactElement => {
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
        menu={(props) => <RemoteSymbolListContextMenu {...props} />}
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
  const createGroup = useCreateGroup();
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
      const key = await createGroup();
      if (key != null) onSelectGroup(key);
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
  const remoteData = Flux.useList((client) => client.groups, {
    name: "symbol groups",
    query: Schematic.Symbol.GROUP_QUERY,
    pageSize: 1000,
    eager: true,
  });
  // A tab strip cannot page, so it reads on until it holds every group.
  useEffect(() => {
    if (remoteData.variant === "success" && !remoteData.complete) remoteData.more();
  }, [remoteData.variant, remoteData.complete]);
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
  onSelectGroup: (group: group.Key) => void;
}

const SearchSymbolListContextMenu = (
  props: StaticSymbolListContextMenuProps,
): ReactElement =>
  z.validate(schematic.symbol.keyZ, props.keys[0]) ? (
    <RemoteSymbolListContextMenu {...props} />
  ) : (
    <StaticSymbolListContextMenu {...props} />
  );

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

const SearchSymbolList = ({
  searchTerm,
  onSelectGroup,
}: SearchSymbolListProps): ReactElement => {
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
  const menuProps = Menu.useContextMenu();
  return (
    <List.Frame<string, Schematic.Node.Spec | schematic.symbol.Symbol>
      data={data}
      getItem={getItem}
      subscribe={subscribe}
    >
      <Menu.ContextMenu
        {...menuProps}
        menu={(props) => (
          <SearchSymbolListContextMenu {...props} onSelectGroup={onSelectGroup} />
        )}
      >
        <List.Scroll
          x
          className={CSS.BE("schematic", "symbols", "group")}
          onContextMenu={menuProps.open}
          wrap
        >
          <List.Items>{searchListItem}</List.Items>
        </List.Scroll>
      </Menu.ContextMenu>
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
  let symbolList = (
    <StaticSymbolList key={groupKey} groupKey={groupKey} onSelectGroup={setGroupKey} />
  );
  if (isRemoteGroup)
    symbolList = <RemoteSymbolList key={groupKey} groupKey={groupKey} />;
  else if (searchMode)
    symbolList = (
      <SearchSymbolList searchTerm={searchTerm} onSelectGroup={setGroupKey} />
    );
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
