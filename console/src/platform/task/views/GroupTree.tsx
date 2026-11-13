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
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree } from "@synnaxlabs/lyra/tree";
import { Channel } from "@synnaxlabs/pluto";
import { deep, id, primitive, type record } from "@synnaxlabs/x";
import { type MouseEvent, type ReactNode, useState, useSyncExternalStore } from "react";

import { Button as PlatformButton } from "@/platform/button";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import {
  isMissing,
  Message,
  MISSING_DESCRIPTIONS,
  NEW_DESCRIPTION,
  STATE_MESSAGES,
  StateIcon,
} from "@/platform/task/ChannelState";
import { EnabledCheckbox } from "@/platform/task/EnabledCheckbox";
import { TareButton } from "@/platform/task/TareButton";
import { READ_CHANNEL_OVERRIDE } from "@/platform/task/types";
import { useIsPreview } from "@/platform/task/useIsPreview";
import { ContextMenu } from "@/platform/task/views/ContextMenu";
import { Panes } from "@/platform/task/views/Panes";

/** A value a group extracts into one channel. */
export interface GroupField {
  key: string;
  disabled: boolean;
  pointer: string;
  channel: number;
  name: string;
}

/** A set of fields read together and stamped by one index. */
export interface Group<F extends GroupField> {
  key: string;
  /** The key of the field whose channel indexes the others, or "" for none. */
  index: string;
  fields: F[];
}

/** A group with no fields beneath it: it is its own field, read whole. */
type Leaf = Omit<GroupField, "pointer">;

/** What the tree reads through the path of any group, with fields or without. */
interface AnyGroup<F extends GroupField> {
  key: string;
  index?: string;
  disabled?: boolean;
  fields?: F[];
  /** A leaf's channel. */
  channel?: number;
}

export interface GroupProps<G> {
  groupKey: string;
  /** The group's form path. */
  path: deep.Path<G>;
}

export interface FieldProps<F> extends GroupProps<F> {
  fieldKey: string;
}

export interface GroupTreeProps<
  G extends Group<F> | Omit<GroupField, "pointer">,
  F extends GroupField,
> {
  /** The form path of the group array. */
  path: deep.Path<G[]>;
  /** Names the groups in the list header: "Endpoints". */
  listTitle: string;
  /** Fills the tree while it holds no groups: "No endpoints". */
  emptyMessage: string;
  /** Fills the details pane while nothing is selected. */
  placeholder: string;
  /** Labels the add button: "Add endpoint". */
  addLabel: string;
  /**
   * Resolves with the groups the add button adds, or none. A group under a key the tree
   * holds replaces that group. A rejection is reported to the user.
   */
  createGroups: () => Promise<G[]>;
  /** A new field's values, without a key. */
  createField: () => Omit<F, "key">;
  /** Names a field whose pointer is empty. */
  emptyPointerLabel?: string;
  /** Lets each group switch off through a `disabled` flag, which it must then hold. */
  groupsSwitchable?: boolean;
  /** The fields with float channels can tare while the task runs. */
  tareable?: boolean;
  /** Names a group in the tree and in its details header. */
  groupLabel: Component.RenderProp<GroupProps<G>>;
  groupDetails: Component.RenderProp<GroupProps<G>>;
  fieldDetails: Component.RenderProp<FieldProps<F>>;
}

type FieldEntry<F> = { kind: "field" } & FieldProps<F>;

type Entry<F> = { kind: "group"; groupKey: string; leaf: boolean } | FieldEntry<F>;

interface Index<F> {
  nodes: Tree.Node[];
  entries: Map<string, Entry<F>>;
  /** Maps the key of each empty group's add field item to the group's key. */
  adders: Map<string, string>;
}

const isBranch = <F extends GroupField>(g: Group<F> | Leaf): g is Group<F> =>
  "fields" in g;

const isShown = (g: { index?: string }, f: GroupField): boolean => f.key !== g.index;

const adderKey = (groupKey: string): string => `${groupKey}/add-field`;

/** Each group key with the keys of its shown fields, or null for a leaf. */
type Structure = [string, string[] | null][];

/**
 * Groups as tree nodes with their fields beneath; the index field stays hidden. Outside
 * preview, an empty group holds one add field item in place of its fields. Edits that
 * keep the structure, such as typing in a field, return the same index.
 */
const useIndex = <G extends Group<F> | Leaf, F extends GroupField>(
  path: deep.Path<G[]>,
  preview: boolean,
): Index<F> => {
  const { bind, get } = Form.useContext();
  const getStructure = () =>
    JSON.stringify(
      get(path).value.map((g): Structure[number] => [
        g.key,
        isBranch<F>(g) ? g.fields.filter((f) => isShown(g, f)).map((f) => f.key) : null,
      ]),
    );
  const structure = useSyncExternalStore(bind, getStructure, getStructure);
  const groups = deep.widen<AnyGroup<F>[]>(path);
  const entries = new Map<string, Entry<F>>();
  const adders = new Map<string, string>();
  const nodes = (JSON.parse(structure) as Structure).map(
    ([groupKey, fieldKeys]): Tree.Node => {
      entries.set(groupKey, { kind: "group", groupKey, leaf: fieldKeys == null });
      if (fieldKeys == null) return { key: groupKey };
      const children: Tree.Node[] = fieldKeys.map((fieldKey) => {
        entries.set(fieldKey, {
          kind: "field",
          groupKey,
          fieldKey,
          path: deep.present(groups[groupKey].fields)[fieldKey],
        });
        return { key: fieldKey };
      });
      if (children.length === 0 && !preview) {
        const key = adderKey(groupKey);
        adders.set(key, groupKey);
        children.push({ key });
      }
      return { key: groupKey, children };
    },
  );
  return { nodes, entries, adders };
};

const entryPath = <G, F>(
  path: deep.Path<G[]>,
  entry: Entry<F>,
): deep.Target<unknown> => (entry.kind === "group" ? path[entry.groupKey] : entry.path);

/** The disabled flag of a field, or of a group that holds one. */
const disabledPath = <F extends GroupField>(
  groups: deep.Path<AnyGroup<F>[]>,
  entry: Entry<F>,
): deep.Path<boolean> =>
  entry.kind === "group"
    ? deep.present(groups[entry.groupKey].disabled)
    : deep.widen<GroupField>(entry.path).disabled;

interface ChannelIconProps {
  channel: deep.Target<number>;
  /** Deploying the task skips the channel. */
  skipped: boolean;
}

/** Marks an item whose channel deploying creates, or whose channel is gone. */
const ChannelIcon = ({ channel, skipped }: ChannelIconProps) => {
  const key = Form.useFieldValue(channel);
  const result = Channel.useResultName(primitive.isNonZero(key) ? { key } : null);
  if (!primitive.isNonZero(key))
    return skipped ? null : (
      <Message
        variant="info"
        message={STATE_MESSAGES.new}
        description={NEW_DESCRIPTION}
      >
        <StateIcon state="new" />
      </Message>
    );
  if (!isMissing(result)) return null;
  return (
    <Message
      variant="warning"
      message={STATE_MESSAGES.missing}
      description={MISSING_DESCRIPTIONS[skipped ? "skips" : "creates"]}
    >
      <StateIcon state="missing" />
    </Message>
  );
};

interface GroupItemProps<G> extends Tree.ItemRenderProps<string> {
  path: deep.Path<G[]>;
  leaf: boolean;
  label: Component.RenderProp<GroupProps<G>>;
  onAddField: (groupKey: string) => void;
  /** Shows the group as switched off. */
  off?: boolean;
  /** Trails the item, such as a switch. */
  children?: ReactNode;
}

const GroupItem = <G extends Group<GroupField> | Leaf>(props: GroupItemProps<G>) => {
  const isPreview = useIsPreview();
  // After the hook: the compiler assumes the label call can change the rest object, so
  // a destructure in the parameters stretches its memo block across the hook.
  const { path, leaf, label, onAddField, off = false, children, ...itemProps } = props;
  const { itemKey } = itemProps;
  const handleAdd = (e: MouseEvent) => {
    e.stopPropagation();
    onAddField(itemKey);
  };
  return (
    <Tree.Item
      {...itemProps}
      className={CSS.cls(
        CSS.B("group-item"),
        leaf && CSS.M("leaf"),
        off && CSS.M("off"),
      )}
    >
      {label({ groupKey: itemKey, path: path[itemKey] })}
      {leaf && (
        <ChannelIcon
          channel={deep.present(
            deep.widen<AnyGroup<GroupField>[]>(path)[itemKey].channel,
          )}
          skipped={off}
        />
      )}
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
      {children}
    </Tree.Item>
  );
};

const SwitchableGroupItem = <G extends Group<GroupField> | Leaf>(
  props: Omit<GroupItemProps<G>, "off" | "children">,
) => {
  const groups = deep.widen<AnyGroup<GroupField>[]>(props.path);
  const disabledField = deep.present(groups[props.itemKey].disabled);
  const disabled = Form.useFieldValue(disabledField);
  return (
    <GroupItem {...props} off={disabled}>
      <EnabledCheckbox path={disabledField} />
    </GroupItem>
  );
};

interface FieldLabelProps extends Pick<Text.TextProps, "level"> {
  path: deep.Target<GroupField>;
  emptyLabel: string;
}

/** Names a field by its pointer, or by emptyLabel while the pointer is empty. */
const FieldLabel = ({ path, level, emptyLabel }: FieldLabelProps) => {
  const pointer = Form.useFieldValue(deep.widen(path).pointer);
  return (
    <Text.Text
      level={level}
      weight={500}
      color={pointer === "" ? 8 : 10}
      overflow="fade"
      className={CSS.B("field-pointer")}
    >
      {pointer === "" ? emptyLabel : pointer}
    </Text.Text>
  );
};

interface FieldItemProps extends Tree.ItemRenderProps<string> {
  path: deep.Target<GroupField>;
  /** The disabled flag of the field's group, which a group need not hold. */
  groupDisabled: deep.Target<boolean | undefined>;
  emptyLabel: string;
  tareable: boolean;
}

const FieldItem = ({
  path,
  groupDisabled,
  emptyLabel,
  tareable,
  ...props
}: FieldItemProps) => {
  const { disabled: disabledField, channel } = deep.widen(path);
  const disabled = Form.useFieldValue(disabledField);
  const groupOff = Form.useFieldValue(groupDisabled, { optional: true }) === true;
  const channelKey = Form.useFieldValue(channel);
  return (
    <Tree.Item
      {...props}
      className={CSS.cls(CSS.B("field-item"), disabled && CSS.M("off"))}
    >
      <FieldLabel path={path} level="small" emptyLabel={emptyLabel} />
      <ChannelIcon channel={channel} skipped={disabled || groupOff} />
      {tareable && <TareButton channel={channelKey} disabled={disabled || groupOff} />}
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

const copyField = <F extends GroupField>(f: F, key: string = id.create()): F => ({
  ...f,
  ...READ_CHANNEL_OVERRIDE,
  key,
});

/** Copies a group under fresh keys, without its channels. */
const copyGroup = <F extends GroupField, G extends Group<F> | Leaf>(g: G): G => {
  if (!isBranch<F>(g)) return { ...g, ...READ_CHANNEL_OVERRIDE, key: id.create() };
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
export const GroupTree = <
  G extends Group<F> | Omit<GroupField, "pointer">,
  F extends GroupField,
>({
  path,
  listTitle,
  emptyMessage,
  placeholder,
  addLabel,
  createGroups,
  createField,
  emptyPointerLabel = "New field",
  groupsSwitchable = false,
  tareable = false,
  groupLabel,
  groupDetails,
  fieldDetails,
}: GroupTreeProps<G, F>): ReactNode => {
  const [selected, setSelected] = useState<string[]>([]);
  const isPreview = useIsPreview();
  const { nodes, entries, adders } = useIndex<G, F>(path, isPreview);
  const groups = deep.widen<AnyGroup<F>[]>(path);
  const ctx = Form.useContext();
  const handleError = Status.useErrorHandler();
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

  const handleAddGroups = () =>
    handleError(async () => {
      const added = await createGroups();
      if (added.length === 0) return;
      const byKey = new Map(added.map((g) => [g.key, g]));
      const kept = ctx.get(path).value.map((g) => {
        const replacement = byKey.get(g.key);
        byKey.delete(g.key);
        return replacement ?? g;
      });
      ctx.set(path, [...kept, ...byKey.values()]);
      setSelected([added[0].key]);
      for (const g of added) if (isBranch<F>(g)) expand(g.key);
    }, `Failed to ${addLabel.toLowerCase()}`);

  const handleAddField = (groupKey: string) => {
    const g = ctx.get(groups[groupKey]).value;
    const fieldsPath = deep.present(groups[groupKey].fields);
    const fields = ctx.get(fieldsPath).value;
    const last = fields.filter((f) => isShown(g, f)).at(-1);
    const field =
      last != null ? copyField(last) : ({ ...createField(), key: id.create() } as F);
    ctx.set(fieldsPath, [...fields, field]);
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
    if (clicked == null) return;
    const entry = entries.get(clicked);
    if (entry?.kind === "group" && !entry.leaf) expand(clicked);
  };

  const handleRemove = (keys: string[]) => {
    const removed = new Set(keys);
    ctx.set(
      path,
      ctx
        .get(path)
        .value.filter((g) => !removed.has(g.key))
        .map((g) =>
          isBranch<F>(g)
            ? { ...g, fields: g.fields.filter((f) => !removed.has(f.key)) }
            : g,
        ),
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
      if (!isBranch<F>(g)) next.push(g);
      else {
        const fields: F[] = [];
        for (const f of g.fields) {
          fields.push(f);
          if (!chosen.has(f.key)) continue;
          const copy = copyField(f);
          if (first == null) first = copy.key;
          fields.push(copy);
        }
        next.push({ ...g, fields });
      }
      if (!chosen.has(g.key)) continue;
      const copy = copyGroup<F, G>(g);
      if (first == null) first = copy.key;
      next.push(copy);
      if (isBranch<F>(copy)) expand(copy.key);
    }
    ctx.set(path, next);
    if (first != null) setSelected([first]);
  };

  const menuProps = Menu.useContextMenu();
  const menuRenderProp = (props: Menu.ContextMenuMenuProps) => {
    const keys = props.keys.filter((k) => !adders.has(k));
    const entry = keys.length === 1 ? entries.get(keys[0]) : undefined;
    const toggles = keys.flatMap((key) => {
      const e = entries.get(key);
      if (e == null || (e.kind === "group" && !groupsSwitchable)) return [];
      const togglePath = disabledPath(groups, e);
      return [{ path: togglePath, disabled: ctx.get(togglePath).value }];
    });
    const setEnabled = (enabled: boolean) =>
      toggles.forEach((t) => ctx.set(t.path, !enabled));
    const tareChannels = keys.flatMap((key) => {
      const e = entries.get(key);
      if (e?.kind !== "field") return [];
      const f = ctx.get(e.path).value;
      const groupOff = ctx.get(groups[e.groupKey]).value.disabled === true;
      return f.disabled || groupOff ? [] : [f.channel];
    });
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
        onEnable={toggles.some((t) => t.disabled) ? () => setEnabled(true) : undefined}
        onDisable={
          toggles.some((t) => !t.disabled) ? () => setEnabled(false) : undefined
        }
        tareChannels={tareable ? tareChannels : undefined}
      />
    );
  };

  const renderItem = ({ key, ...p }: Tree.ItemRenderProps<string>) => {
    if (adders.has(p.itemKey)) return <AdderItem key={key} {...p} />;
    const entry = entries.get(p.itemKey);
    if (entry?.kind === "field")
      return (
        <FieldItem
          key={key}
          {...p}
          path={entry.path}
          groupDisabled={deep.widen<AnyGroup<F>[]>(path)[entry.groupKey].disabled}
          emptyLabel={emptyPointerLabel}
          tareable={tareable}
        />
      );
    const itemProps = {
      ...p,
      path,
      leaf: entry?.leaf ?? false,
      label: groupLabel,
      onAddField: handleAddField,
    };
    return groupsSwitchable ? (
      <SwitchableGroupItem key={key} {...itemProps} />
    ) : (
      <GroupItem key={key} {...itemProps} />
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
              emptyContent={<Empty.Action message={emptyMessage} />}
              allowNone={false}
              autoSelectOnNone
            >
              {renderItem}
            </Tree.Tree>
          </Menu.ContextMenu>
          {!isPreview && (
            <PlatformButton.CreateListItem size="small" onClick={handleAddGroups}>
              {addLabel}
            </PlatformButton.CreateListItem>
          )}
        </>
      }
      detailsPath={current != null ? entryPath(path, current) : null}
      title={
        current?.kind === "group" ? (
          groupLabel({ groupKey: current.groupKey, path: path[current.groupKey] })
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
        groupDetails({ groupKey: current.groupKey, path: path[current.groupKey] })
      ) : (
        fieldDetails(current)
      )}
    </Panes>
  );
};
