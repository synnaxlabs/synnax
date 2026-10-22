// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/views/GroupTree.css";

import { Button } from "@synnaxlabs/lyra/button";
import { type Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree } from "@synnaxlabs/lyra/tree";
import { id, type record } from "@synnaxlabs/x";
import { type MouseEvent, type ReactNode, useCallback, useMemo, useState } from "react";

import { Button as PlatformButton } from "@/platform/button";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { EnabledCheckbox } from "@/platform/task/EnabledCheckbox";
import { useIsPreview } from "@/platform/task/Form";
import { READ_CHANNEL_OVERRIDE } from "@/platform/task/types";
import { ContextMenu } from "@/platform/task/views/ContextMenu";
import { Panes } from "@/platform/task/views/Panes";
import { type Drop, useDrop } from "@/platform/task/views/useDrop";

/** A value a group extracts into one channel. */
export interface GroupField {
  key: string;
  disabled: boolean;
  pointer: string;
}

/**
 * A set of fields read together and stamped by one index. A group with a disabled
 * flag can be switched off.
 */
export interface Group<F extends GroupField> {
  key: string;
  disabled?: boolean;
  /** The key of the field whose channel indexes the others, or "" for none. */
  index: string;
  fields: F[];
}

/** A group read straight into its own channel, with no fields beneath it. */
export interface LeafGroup {
  key: string;
  disabled?: boolean;
  channel: number;
  name: string;
  /** The key of the channel that indexes the group's channel. */
  index: number;
}

/** A kind of group the user adds through its own add button. */
export interface GroupKind<G> {
  /** Names the kind for its add button: "endpoint". */
  name: string;
  /** Returns a new group under a fresh key. */
  create: () => G;
}

export interface GroupProps {
  groupKey: string;
  /** The group's form path. */
  path: string;
}

export interface FieldProps extends GroupProps {
  fieldKey: string;
}

export interface GroupTreeProps<G extends Group<F> | LeafGroup, F extends GroupField> {
  /** The form path of the group array. */
  path?: string;
  /** Names the groups in the list header: "Endpoints". */
  listTitle: string;
  /** Fills the tree while it holds no groups: "No endpoints". */
  emptyMessage: string;
  /** Fills the details pane while nothing is selected. */
  placeholder: string;
  groupKinds: GroupKind<G>[];
  /** A new field's values, without a key. */
  createField: () => Omit<F, "key">;
  /** Names a field whose pointer is empty. */
  emptyPointerLabel?: string;
  /** Names a group in its tree row and details header. */
  groupLabel: Component.RenderProp<GroupProps>;
  groupDetails: Component.RenderProp<GroupProps>;
  fieldDetails: Component.RenderProp<FieldProps>;
  /** Adds the groups built from items dragged onto the tree. */
  drop?: Drop<G>;
}

type Entry =
  { kind: "group"; groupKey: string; leaf: boolean } | ({ kind: "field" } & FieldProps);

interface Index {
  nodes: Tree.Node[];
  entries: Map<string, Entry>;
}

const isBranch = <F extends GroupField>(g: Group<F> | LeafGroup): g is Group<F> =>
  "fields" in g;

const isShown = <F extends GroupField>(g: Group<F>, f: F): boolean => f.key !== g.index;

/** Groups as tree nodes with their fields beneath; the index field stays hidden. */
const useIndex = <F extends GroupField>(path: string): Index => {
  // The form writes child paths into the same array, so the array's identity never
  // changes; the field state's does on every write beneath it.
  const state = Form.useFieldState<(Group<F> | LeafGroup)[]>(path);
  return useMemo(() => {
    const entries = new Map<string, Entry>();
    const nodes = state.value.map((g): Tree.Node => {
      const leaf = !isBranch<F>(g);
      entries.set(g.key, { kind: "group", groupKey: g.key, leaf });
      if (!isBranch<F>(g)) return { key: g.key };
      const groupPath = `${path}.${g.key}`;
      const children = g.fields
        .filter((f) => isShown(g, f))
        .map((f) => {
          entries.set(f.key, {
            kind: "field",
            groupKey: g.key,
            fieldKey: f.key,
            path: `${groupPath}.fields.${f.key}`,
          });
          return { key: f.key };
        });
      return { key: g.key, children };
    });
    return { nodes, entries };
  }, [path, state]);
};

const entryPath = (path: string, entry: Entry): string =>
  entry.kind === "group" ? `${path}.${entry.groupKey}` : entry.path;

interface GroupItemProps extends Tree.ItemRenderProps<string> {
  path: string;
  leaf: boolean;
  label: Component.RenderProp<GroupProps>;
  onAddField: (groupKey: string) => void;
}

const GroupItem = ({ path, leaf, label, onAddField, ...props }: GroupItemProps) => {
  const { itemKey } = props;
  const groupPath = `${path}.${itemKey}`;
  // Null for a group without a disabled flag.
  const disabled = Form.useFieldValue<boolean>(`${groupPath}.disabled`, {
    optional: true,
  });
  const isPreview = useIsPreview();
  const handleAdd = useCallback(
    (e: MouseEvent) => {
      e.stopPropagation();
      onAddField(itemKey);
    },
    [onAddField, itemKey],
  );
  return (
    <Tree.Item
      {...props}
      className={CSS.cls(
        CSS.B("group-item"),
        leaf && CSS.M("leaf"),
        disabled === true && CSS.M("off"),
      )}
    >
      {label({ groupKey: itemKey, path: groupPath })}
      {!leaf && !isPreview && (
        <Button.Button
          onClick={handleAdd}
          variant="text"
          size="tiny"
          tooltip="Add field"
          aria-label="Add field"
          tooltipLocation="right"
          className={CSS.BE("group-item", "add")}
        >
          <Icon.Add />
        </Button.Button>
      )}
      <EnabledCheckbox path={`${groupPath}.disabled`} />
    </Tree.Item>
  );
};

interface FieldLabelProps extends Pick<Text.TextProps, "level"> {
  path: string;
  emptyLabel: string;
}

/** Names a field by its pointer, or by emptyLabel while the pointer is empty. */
const FieldLabel = ({ path, level, emptyLabel }: FieldLabelProps) => {
  const pointer = Form.useFieldValue<string>(`${path}.pointer`);
  return (
    <Text.Text
      level={level}
      weight={500}
      color={pointer === "" ? 8 : 10}
      overflow="ellipsis"
      className={CSS.B("field-pointer")}
    >
      {pointer === "" ? emptyLabel : pointer}
    </Text.Text>
  );
};

interface FieldItemProps extends Tree.ItemRenderProps<string> {
  path: string;
  emptyLabel: string;
}

const FieldItem = ({ path, emptyLabel, ...props }: FieldItemProps) => {
  const disabled = Form.useFieldValue<boolean>(`${path}.disabled`);
  return (
    <Tree.Item
      {...props}
      className={CSS.cls(CSS.B("field-item"), disabled && CSS.M("off"))}
    >
      <FieldLabel path={path} level="small" emptyLabel={emptyLabel} />
      <EnabledCheckbox path={`${path}.disabled`} />
    </Tree.Item>
  );
};

const TREE_ITEM_HEIGHT = 36;

const copyField = <F extends GroupField>(f: F, key: string = id.create()): F => ({
  ...f,
  ...READ_CHANNEL_OVERRIDE,
  key,
});

/** Copies a group under fresh keys, without its channels. */
const copyGroup = <F extends GroupField, G extends Group<F> | LeafGroup>(g: G): G => {
  if (!isBranch<F>(g))
    return { ...g, ...READ_CHANNEL_OVERRIDE, key: id.create(), index: 0 };
  const keys = new Map(g.fields.map((f) => [f.key, id.create()]));
  return {
    ...g,
    key: id.create(),
    index: keys.get(g.index) ?? g.index,
    fields: g.fields.map((f) => copyField(f, keys.get(f.key))),
  };
};

/**
 * Lists groups with their fields beneath, beside the details of the selected one. A
 * group's index field is hidden from the tree; the group details edit it.
 */
export const GroupTree = <G extends Group<F> | LeafGroup, F extends GroupField>({
  path = "config.groups",
  listTitle,
  emptyMessage,
  placeholder,
  groupKinds,
  createField,
  emptyPointerLabel = "New field",
  groupLabel,
  groupDetails,
  fieldDetails,
  drop,
}: GroupTreeProps<G, F>): ReactNode => {
  const [selected, setSelected] = useState<string[]>([]);
  const { nodes, entries } = useIndex<F>(path);
  const ctx = Form.useContext();
  const isPreview = useIsPreview();
  const [initialExpanded] = useState(() => nodes.map(({ key }) => key));
  // A fold that hides the selected field moves the selection up to its group.
  const handleExpand = useCallback(
    ({ action, clicked }: Tree.HandleExpandProps<string>) => {
      if (action !== "contract") return;
      const hidesSelection = selected.some((k) => {
        const entry = entries.get(k);
        return entry?.kind === "field" && entry.groupKey === clicked;
      });
      if (hidesSelection) setSelected([clicked]);
    },
    [selected, entries],
  );
  const treeProps = Tree.use({
    nodes,
    selected,
    onSelectedChange: setSelected,
    initialExpanded,
    onExpand: handleExpand,
    toggleOn: "caret",
  });
  const { expand, shape } = treeProps;

  const handleSelect = useCallback<
    Tree.TreeProps<string, record.Keyed<string>>["onSelect"]
  >(
    (keys, { clicked }) => {
      setSelected(keys);
      if (clicked == null) return;
      const entry = entries.get(clicked);
      if (entry?.kind === "group" && !entry.leaf) expand(clicked);
    },
    [entries, expand],
  );

  const handleAddGroups = useCallback(
    (added: G[]) => {
      const groups = ctx.get<G[]>(path).value;
      ctx.set(path, [...groups, ...added]);
      setSelected([added[0].key]);
      for (const g of added) if (isBranch<F>(g)) expand(g.key);
    },
    [ctx, expand, path],
  );
  const dropProps = useDrop({ drop, onAdd: handleAddGroups });

  const handleAddField = useCallback(
    (groupKey: string) => {
      const g = ctx.get<Group<F>>(`${path}.${groupKey}`).value;
      const shown = g.fields.filter((f) => isShown(g, f));
      const last = shown.at(-1);
      const field =
        last != null ? copyField(last) : ({ ...createField(), key: id.create() } as F);
      ctx.set(`${path}.${groupKey}.fields`, [...g.fields, field]);
      setSelected([field.key]);
      expand(groupKey);
    },
    [ctx, expand, createField, path],
  );

  const handleRemove = useCallback(
    (keys: string[]) => {
      const removed = new Set(keys);
      const groups = ctx.get<G[]>(path).value;
      ctx.set(
        path,
        groups
          .filter((g) => !removed.has(g.key))
          .map((g) =>
            isBranch<F>(g)
              ? { ...g, fields: g.fields.filter((f) => !removed.has(f.key)) }
              : g,
          ),
      );
      // Selection moves to the nearest survivor in the visible list, below first.
      const visible = shape.keys;
      const first = visible.findIndex((k) => removed.has(k));
      const after = visible.slice(first).find((k) => !removed.has(k));
      const before = visible
        .slice(0, Math.max(first, 0))
        .reverse()
        .find((k) => !removed.has(k));
      const next = after ?? before;
      setSelected(next == null ? [] : [next]);
    },
    [ctx, shape, path],
  );

  const handleDuplicate = useCallback(
    (keys: string[]) => {
      const chosen = new Set(keys);
      const groups = ctx.get<G[]>(path).value;
      const next: G[] = [];
      let first: string | null = null;
      for (const g of groups) {
        if (!isBranch<F>(g)) next.push(g);
        else {
          const fields: F[] = [];
          for (const f of g.fields) {
            fields.push(f);
            if (!chosen.has(f.key)) continue;
            const copy = copyField(f);
            first ??= copy.key;
            fields.push(copy);
          }
          next.push({ ...g, fields });
        }
        if (!chosen.has(g.key)) continue;
        const copy = copyGroup<F, G>(g);
        first ??= copy.key;
        next.push(copy);
        if (isBranch<F>(copy)) expand(copy.key);
      }
      ctx.set(path, next);
      if (first != null) setSelected([first]);
    },
    [ctx, expand, path],
  );

  const menuProps = Menu.useContextMenu();
  const menuRenderProp = useCallback(
    ({ keys }: Menu.ContextMenuMenuProps) => {
      const entry = keys.length === 1 ? entries.get(keys[0]) : undefined;
      // A group without a disabled flag is dropped, never switched off.
      const toggles = keys.flatMap((key) => {
        const e = entries.get(key);
        if (e == null) return [];
        const togglePath = `${entryPath(path, e)}.disabled`;
        const state = ctx.get<boolean>(togglePath, { optional: true });
        return state == null ? [] : [{ path: togglePath, disabled: state.value }];
      });
      const setEnabled = (enabled: boolean) =>
        toggles.forEach((t) => ctx.set(t.path, !enabled));
      return (
        <ContextMenu
          keys={keys}
          onRemove={handleRemove}
          onDuplicate={handleDuplicate}
          onAddField={
            entry?.kind === "group" && !entry.leaf
              ? () => handleAddField(entry.groupKey)
              : undefined
          }
          onEnable={
            toggles.some((t) => t.disabled) ? () => setEnabled(true) : undefined
          }
          onDisable={
            toggles.some((t) => !t.disabled) ? () => setEnabled(false) : undefined
          }
        />
      );
    },
    [ctx, entries, path, handleRemove, handleDuplicate, handleAddField],
  );

  const renderItem = useCallback(
    ({ key, ...p }: Tree.ItemRenderProps<string>) => {
      const entry = entries.get(p.itemKey);
      if (entry?.kind === "field")
        return (
          <FieldItem
            key={key}
            {...p}
            path={entry.path}
            emptyLabel={emptyPointerLabel}
          />
        );
      return (
        <GroupItem
          key={key}
          {...p}
          path={path}
          leaf={entry?.leaf ?? false}
          label={groupLabel}
          onAddField={handleAddField}
        />
      );
    },
    [entries, handleAddField, groupLabel, emptyPointerLabel, path],
  );

  const current = selected.length > 0 ? entries.get(selected[0]) : undefined;

  return (
    <Panes
      listTitle={listTitle}
      list={
        <>
          <Menu.ContextMenu {...menuProps} menu={menuRenderProp}>
            <Tree.Tree<string, record.Keyed<string>>
              {...treeProps}
              {...dropProps}
              onSelect={handleSelect}
              itemHeight={TREE_ITEM_HEIGHT}
              className={CSS.cls(menuProps.className, CSS.B("group-tree"))}
              onContextMenu={menuProps.open}
              emptyContent={<Empty.Action message={emptyMessage} />}
              allowNone={false}
              autoSelectOnNone
            >
              {renderItem}
            </Tree.Tree>
          </Menu.ContextMenu>
          {!isPreview &&
            groupKinds.map(({ name, create }) => (
              <PlatformButton.CreateListItem
                key={name}
                size="small"
                onClick={() => handleAddGroups([create()])}
              >
                {`Add ${name}`}
              </PlatformButton.CreateListItem>
            ))}
        </>
      }
      detailsPath={current != null ? entryPath(path, current) : null}
      title={
        current?.kind === "group" ? (
          groupLabel({
            groupKey: current.groupKey,
            path: `${path}.${current.groupKey}`,
          })
        ) : current?.kind === "field" ? (
          <FieldLabel path={current.path} level="p" emptyLabel={emptyPointerLabel} />
        ) : undefined
      }
    >
      {current == null ? (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text status="disabled">{placeholder}</Text.Text>
        </Flex.Box>
      ) : current.kind === "group" ? (
        groupDetails({
          groupKey: current.groupKey,
          path: `${path}.${current.groupKey}`,
        })
      ) : (
        fieldDetails(current)
      )}
    </Panes>
  );
};
