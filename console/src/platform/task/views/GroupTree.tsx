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
import { deep, id, type record } from "@synnaxlabs/x";
import { type MouseEvent, type ReactNode, useState } from "react";

import { Button as PlatformButton } from "@/platform/button";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { EnabledCheckbox } from "@/platform/task/EnabledCheckbox";
import { READ_CHANNEL_OVERRIDE } from "@/platform/task/types";
import { useIsPreview } from "@/platform/task/useIsPreview";
import { ContextMenu } from "@/platform/task/views/ContextMenu";
import { Panes } from "@/platform/task/views/Panes";

/** A value a group extracts into one channel. */
export interface GroupField {
  key: string;
  disabled: boolean;
  pointer: string;
}

/** A set of fields read together and stamped by one index. */
export interface Group<F extends GroupField> {
  key: string;
  /** The key of the field whose channel indexes the others, or "" for none. */
  index: string;
  fields: F[];
}

export interface GroupProps<G> {
  groupKey: string;
  /** The group's form path. */
  path: deep.Path<G>;
}

export interface FieldProps<F> extends GroupProps<F> {
  fieldKey: string;
}

export interface GroupTreeProps<G extends Group<F>, F extends GroupField> {
  /** The form path of the group array. */
  path: deep.Path<G[]>;
  /** Names the groups in the list header: "Endpoints". */
  listTitle: string;
  /** Names one group in lowercase for the add button and empty state: "endpoint". */
  groupName: string;
  /** Fills the details pane while nothing is selected. */
  placeholder: string;
  /** A new group's values, without a key. */
  createGroup: () => Omit<G, "key">;
  /** A new field's values, without a key. */
  createField: () => Omit<F, "key">;
  /** Names a group in its tree row and details header. */
  groupLabel: Component.RenderProp<GroupProps<G>>;
  groupDetails: Component.RenderProp<GroupProps<G>>;
  fieldDetails: Component.RenderProp<FieldProps<F>>;
}

type FieldEntry<F> = { kind: "field" } & FieldProps<F>;

type Entry<F> = { kind: "group"; groupKey: string } | FieldEntry<F>;

interface Index<F> {
  nodes: Tree.Node[];
  entries: Map<string, Entry<F>>;
  /** Maps the key of each empty group's add field item to the group's key. */
  adders: Map<string, string>;
}

const isShown = <F extends GroupField>(g: Group<F>, f: F): boolean => f.key !== g.index;

const adderKey = (groupKey: string): string => `${groupKey}/add-field`;

/**
 * Groups as tree nodes with their fields beneath; the index field stays hidden. Outside
 * preview, an empty group holds one add field item in place of its fields.
 */
const useIndex = <G extends Group<F>, F extends GroupField>(
  path: deep.Path<G[]>,
  preview: boolean,
): Index<F> => {
  // The form writes child paths into the same array, so the array's identity never
  // changes; the field state's does on every write beneath it.
  const state = Form.useFieldState(path);
  const groups = deep.widen<Group<F>[]>(path);
  const entries = new Map<string, Entry<F>>();
  const adders = new Map<string, string>();
  const nodes = state.value.map((g) => {
    entries.set(g.key, { kind: "group", groupKey: g.key });
    const children = g.fields
      .filter((f) => isShown(g, f))
      .map((f) => {
        entries.set(f.key, {
          kind: "field",
          groupKey: g.key,
          fieldKey: f.key,
          path: groups[g.key].fields[f.key],
        });
        return { key: f.key };
      });
    if (children.length === 0 && !preview) {
      const key = adderKey(g.key);
      adders.set(key, g.key);
      children.push({ key });
    }
    return { key: g.key, children };
  });
  return { nodes, entries, adders };
};

const entryPath = <G, F>(
  path: deep.Path<G[]>,
  entry: Entry<F>,
): deep.Target<unknown> => (entry.kind === "group" ? path[entry.groupKey] : entry.path);

interface GroupItemProps<G> extends Tree.ItemRenderProps<string> {
  path: deep.Path<G[]>;
  label: Component.RenderProp<GroupProps<G>>;
  onAddField: (groupKey: string) => void;
}

const GroupItem = <G,>(props: GroupItemProps<G>) => {
  const isPreview = useIsPreview();
  // After the hook: the compiler assumes the label call can change the rest object, so
  // a destructure in the parameters stretches its memo block across the hook.
  const { path, label, onAddField, ...itemProps } = props;
  const { itemKey } = itemProps;
  const handleAdd = (e: MouseEvent) => {
    e.stopPropagation();
    onAddField(itemKey);
  };
  return (
    <Tree.Item {...itemProps} className={CSS.B("group-item")}>
      {label({ groupKey: itemKey, path: path[itemKey] })}
      {!isPreview && (
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
    </Tree.Item>
  );
};

interface FieldLabelProps extends Pick<Text.TextProps, "level"> {
  path: deep.Target<GroupField>;
}

/** Names a field by its pointer; an empty pointer reads as a new field. */
const FieldLabel = ({ path, level }: FieldLabelProps) => {
  const pointer = Form.useFieldValue(deep.widen(path).pointer);
  return (
    <Text.Text
      level={level}
      weight={500}
      color={pointer === "" ? 8 : 10}
      overflow="fade"
      className={CSS.B("field-pointer")}
    >
      {pointer === "" ? "New field" : pointer}
    </Text.Text>
  );
};

interface FieldItemProps extends Tree.ItemRenderProps<string> {
  path: deep.Target<GroupField>;
}

const FieldItem = ({ path, ...props }: FieldItemProps) => {
  const { disabled: disabledField } = deep.widen(path);
  const disabled = Form.useFieldValue(disabledField);
  return (
    <Tree.Item
      {...props}
      className={CSS.cls(CSS.B("field-item"), disabled && CSS.M("off"))}
    >
      <FieldLabel path={path} level="small" />
      <EnabledCheckbox path={disabledField} />
    </Tree.Item>
  );
};

/** Stands in for an empty group's fields; choosing it adds the first one. */
const AdderItem = (props: Tree.ItemRenderProps<string>) => (
  <Tree.Item {...props} level="small" textColor={9} className={CSS.B("adder-item")}>
    <Icon.Add />
    Add field
  </Tree.Item>
);

const TREE_ITEM_HEIGHT = 36;

const disabledPath = <F extends GroupField>({ path }: FieldEntry<F>) =>
  deep.widen<GroupField>(path).disabled;

const copyField = <F extends GroupField>(f: F, key: string = id.create()): F => ({
  ...f,
  ...READ_CHANNEL_OVERRIDE,
  key,
});

/**
 * Lists groups with their fields beneath, beside the details of the selected one. A
 * group's index field is hidden from the tree; the group details edit it.
 */
export const GroupTree = <G extends Group<F>, F extends GroupField>({
  path,
  listTitle,
  groupName,
  placeholder,
  createGroup,
  createField,
  groupLabel,
  groupDetails,
  fieldDetails,
}: GroupTreeProps<G, F>): ReactNode => {
  const [selected, setSelected] = useState<string[]>([]);
  const isPreview = useIsPreview();
  const { nodes, entries, adders } = useIndex<G, F>(path, isPreview);
  const groups = deep.widen<Group<F>[]>(path);
  const ctx = Form.useContext();
  const [initialExpanded] = useState(() => nodes.map(({ key }) => key));
  // A fold that hides the selected field moves the selection up to its group.
  const handleExpand = ({ action, clicked }: Tree.HandleExpandProps<string>) => {
    if (action !== "contract") return;
    const hidesSelection = selected.some((k) => {
      const entry = entries.get(k);
      return entry?.kind === "field" && entry.groupKey === clicked;
    });
    if (hidesSelection) setSelected([clicked]);
  };
  const treeProps = Tree.use({
    nodes,
    selected,
    onSelectedChange: setSelected,
    initialExpanded,
    onExpand: handleExpand,
    toggleOn: "caret",
  });
  const { expand, shape } = treeProps;

  const handleAddGroup = () => {
    const g = { ...createGroup(), key: id.create() } as G;
    ctx.set(path, [...ctx.get(path).value, g]);
    setSelected([g.key]);
    expand(g.key);
  };

  const handleAddField = (groupKey: string) => {
    const g = ctx.get(groups[groupKey]).value;
    const shown = g.fields.filter((f) => isShown(g, f));
    const last = shown.at(-1);
    const field =
      last != null ? copyField(last) : ({ ...createField(), key: id.create() } as F);
    ctx.set(groups[groupKey].fields, [...g.fields, field]);
    setSelected([field.key]);
    expand(groupKey);
  };

  // An add field item is never selected; choosing it adds a field instead.
  const handleSelect: Tree.TreeProps<string, record.Keyed<string>>["onSelect"] = (
    keys,
    { clicked },
  ) => {
    const adderGroup = clicked == null ? undefined : adders.get(clicked);
    if (adderGroup != null) return handleAddField(adderGroup);
    setSelected(keys.filter((k) => !adders.has(k)));
    if (clicked != null && entries.get(clicked)?.kind === "group") expand(clicked);
  };

  const handleRemove = (keys: string[]) => {
    const removed = new Set(keys);
    ctx.set(
      path,
      ctx
        .get(path)
        .value.filter((g) => !removed.has(g.key))
        .map((g) => ({ ...g, fields: g.fields.filter((f) => !removed.has(f.key)) })),
    );
    // Selection moves to the nearest survivor in the visible list, below first.
    const visible = shape.keys.filter((k) => !adders.has(k));
    const first = visible.findIndex((k) => removed.has(k));
    const after = visible.slice(first).find((k) => !removed.has(k));
    const before = visible
      .slice(0, Math.max(first, 0))
      .reverse()
      .find((k) => !removed.has(k));
    const next = after ?? before;
    setSelected(next == null ? [] : [next]);
  };

  const handleDuplicate = (keys: string[]) => {
    const chosen = new Set(keys);
    const next: G[] = [];
    let first: string | null = null;
    for (const g of ctx.get(path).value) {
      const fields: F[] = [];
      for (const f of g.fields) {
        fields.push(f);
        if (!chosen.has(f.key)) continue;
        const copy = copyField(f);
        if (first == null) first = copy.key;
        fields.push(copy);
      }
      next.push({ ...g, fields });
      if (!chosen.has(g.key)) continue;
      const copiedKeys = new Map(g.fields.map((f) => [f.key, id.create()]));
      const copy: G = {
        ...g,
        key: id.create(),
        index: copiedKeys.get(g.index) ?? g.index,
        fields: g.fields.map((f) => copyField(f, copiedKeys.get(f.key))),
      };
      if (first == null) first = copy.key;
      next.push(copy);
      expand(copy.key);
    }
    ctx.set(path, next);
    if (first != null) setSelected([first]);
  };

  // Only fields carry a disabled flag; a group is dropped, never switched off.
  const handleSetEnabled = (keys: string[], enabled: boolean) => {
    for (const key of keys) {
      const entry = entries.get(key);
      if (entry?.kind === "field") ctx.set(disabledPath(entry), !enabled);
    }
  };

  const menuProps = Menu.useContextMenu();
  const menuRenderProp = (props: Menu.ContextMenuMenuProps) => {
    const keys = props.keys.filter((k) => !adders.has(k));
    const entry = keys.length === 1 ? entries.get(keys[0]) : undefined;
    const fields = keys
      .map((key) => entries.get(key))
      .filter((e) => e?.kind === "field");
    const disabledOf = (e: FieldEntry<F>) => ctx.get(disabledPath(e)).value;
    return (
      <ContextMenu
        keys={keys}
        onRemove={handleRemove}
        onDuplicate={handleDuplicate}
        onAddField={
          entry?.kind === "group" ? () => handleAddField(entry.groupKey) : undefined
        }
        onEnable={
          fields.some(disabledOf) ? () => handleSetEnabled(keys, true) : undefined
        }
        onDisable={
          fields.some((e) => !disabledOf(e))
            ? () => handleSetEnabled(keys, false)
            : undefined
        }
      />
    );
  };

  const renderItem = ({ key, ...p }: Tree.ItemRenderProps<string>) => {
    if (adders.has(p.itemKey)) return <AdderItem key={key} {...p} />;
    const entry = entries.get(p.itemKey);
    if (entry?.kind === "field")
      return <FieldItem key={key} {...p} path={entry.path} />;
    return (
      <GroupItem
        key={key}
        {...p}
        path={path}
        label={groupLabel}
        onAddField={handleAddField}
      />
    );
  };

  const current = selected.length > 0 ? entries.get(selected[0]) : undefined;

  return (
    <Panes
      listTitle={listTitle}
      list={
        <>
          <Menu.ContextMenu {...menuProps} menu={menuRenderProp}>
            <Tree.Tree<string, record.Keyed<string>>
              {...treeProps}
              onSelect={handleSelect}
              itemHeight={TREE_ITEM_HEIGHT}
              className={CSS.cls(menuProps.className, CSS.B("group-tree"))}
              onContextMenu={menuProps.open}
              emptyContent={<Empty.Action message={`No ${groupName}s`} />}
              allowNone={false}
              autoSelectOnNone
            >
              {renderItem}
            </Tree.Tree>
          </Menu.ContextMenu>
          {!isPreview && (
            <PlatformButton.CreateListItem size="small" onClick={handleAddGroup}>
              {`Add ${groupName}`}
            </PlatformButton.CreateListItem>
          )}
        </>
      }
      detailsPath={current != null ? entryPath(path, current) : null}
      title={
        current?.kind === "group" ? (
          groupLabel({ groupKey: current.groupKey, path: path[current.groupKey] })
        ) : current?.kind === "field" ? (
          <FieldLabel path={current.path} level="p" />
        ) : undefined
      }
    >
      {current == null ? (
        <Flex.Box y grow align="center" justify="center">
          <Text.Text status="disabled">{placeholder}</Text.Text>
        </Flex.Box>
      ) : current.kind === "group" ? (
        groupDetails({ groupKey: current.groupKey, path: path[current.groupKey] })
      ) : (
        fieldDetails(current)
      )}
    </Panes>
  );
};
