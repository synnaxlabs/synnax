// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import type { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree } from "@synnaxlabs/lyra/tree";
import { deep, type record, unique } from "@synnaxlabs/x";
import {
  type Dispatch,
  type FC,
  type ReactElement,
  type SetStateAction,
  useState,
  useSyncExternalStore,
} from "react";

import { createJSONField } from "@/platform/icd/Fields";
import { fieldsOf, formatOf, useFormats } from "@/platform/icd/Formats";
import { fieldsPath, type TreeField, treeFieldOf } from "@/platform/icd/read";
import { fieldKey } from "@/platform/icd/tree/nodes";
import type { FieldDetailsProps } from "@/platform/icd/write/Details";
import { Task } from "@/platform/task";

/** A target of a write task, in the task's form. */
export interface Target extends record.Keyed<string> {
  disabled: boolean;
}

/** The message that a target sends. */
export interface TargetMessage {
  icd: icd.Key;
  message: icd.MessageKey;
  /** The path of the target's bindings in the task's form. */
  bindings: deep.Path<icd.WriteBinding[]>;
}

/** A node of the tree that is not a group. */
export type Node =
  | { kind: "target"; target: string }
  | { kind: "field"; target: string; field: TreeField };

export interface GroupItemProps extends Tree.ItemRenderProps<string> {
  /** The name of the group, as its targets give it. */
  name: string;
}

/** The groups of the targets of a task, such as the topics of Kafka records. */
export interface Groups<T extends Target> {
  /** @returns the name of the group of a target. Targets of one name share it. */
  of: (target: T) => string;
  Item: FC<GroupItemProps>;
}

export interface UseTreeArgs<T extends Target> {
  /** The targets of the task in its form. */
  path: deep.Target<T[]>;
  /** @returns the message that a target sends, or null for a target with no fields. */
  messageOf: (target: T) => TargetMessage | null;
  /** Puts the targets in groups. Omitted, the targets are the roots of the tree. */
  groups?: Groups<T>;
  TargetItem: FC<Tree.ItemRenderProps<string>>;
  /** @returns a copy of a target with a new key. Omitted, the menu has no Duplicate. */
  duplicate?: (target: T) => T;
  /** The key of each selected tree item. */
  selected: string[];
  onSelect: Dispatch<SetStateAction<string[]>>;
  expanded: string[];
  onExpand: Dispatch<SetStateAction<string[]>>;
}

export interface UseTreeReturn {
  tree: Tree.UseReturn<string>;
  handleSelect: Tree.TreeProps<string, record.Keyed<string>>["onSelect"];
  contextMenu: (props: Menu.ContextMenuMenuProps) => ReactElement;
  renderItem: (props: Tree.ItemRenderProps<string>) => ReactElement;
  /** The first selected node, unless it is a group. */
  current?: Node;
  /** The name of the first selected node, when it is a group. */
  currentGroup?: string;
  /**
   * Adds a field to the JSON message of a target and to its ICD, and selects it. The
   * Suit of the target binds it.
   */
  addField: (target: string) => void;
  /**
   * @returns how the details of a field edit it: the name of the field that addField
   * added last takes the focus, and a rename of a field that addField added moves its
   * pointer until the user edits the pointer.
   */
  fieldProps: (field: icd.FieldKey) => FieldProps;
}

export type FieldProps = Pick<
  FieldDetailsProps,
  "autoFocus" | "pointerLinked" | "onPointerChange"
>;

const groupKey = (name: string): string => `group/${name}`;

/** @returns the keys that show the fields of targets: each target, and its group. */
export const expandedOf = <T extends Target>(
  targets: T[],
  groups?: Groups<T>,
): string[] =>
  unique.unique(
    targets.flatMap((t) =>
      groups == null ? [t.key] : [groupKey(groups.of(t)), t.key],
    ),
  );

/** Each target key, the name of its group, and its message and fields, or null. */
type Structure = [string, string | null, [icd.MessageKey, TreeField[]] | null][];

interface FieldItemProps extends Tree.ItemRenderProps<string> {
  bindingsPath: deep.Path<icd.WriteBinding[]>;
  disabledPath: deep.Path<boolean>;
  field: TreeField;
}

const FieldItem = ({
  bindingsPath: path,
  disabledPath,
  field,
  ...props
}: FieldItemProps) => {
  const bindings = PForm.useFieldValue(path);
  const skipped = PForm.useFieldValue(disabledPath);
  const i = bindings.findIndex((b) => b.field === field.key);
  const source = i < 0 ? "default" : bindings[i].type;
  return (
    <Task.Views.TreeFieldItem
      {...props}
      label={field.name}
      emptyLabel="Unnamed field"
      channel={
        source === "channel" ? deep.variant(path[i], "type", "channel").channel : null
      }
      off={false}
      skipped={skipped}
    >
      {source !== "channel" && (
        <Text.Text level="small" color={8}>
          {source === "value" ? "Value" : "Default"}
        </Text.Text>
      )}
    </Task.Views.TreeFieldItem>
  );
};

/**
 * The tree of the targets of a write task, with every field of the message that each
 * sends, under the group of each target when it has one. It edits the ICDs of the
 * targets through their open forms, so call it inside a WithFormats that opens them.
 */
export const useTree = <T extends Target>({
  path,
  messageOf,
  groups,
  TargetItem,
  duplicate,
  selected,
  onSelect,
  expanded,
  onExpand,
}: UseTreeArgs<T>): UseTreeReturn => {
  const ctx = PForm.useContext();
  const formats = useFormats();
  const [added, setAdded] = useState<icd.FieldKey>();
  const [linked, setLinked] = useState<icd.FieldKey[]>([]);
  const subscribe = (onChange: () => void) => {
    const unbinds = [
      ctx.bind(onChange),
      ...[...formats.values()].map((f) => f.bind(onChange)),
    ];
    return () => unbinds.forEach((unbind) => unbind());
  };
  const getStructure = () =>
    JSON.stringify(
      ctx.get(path).value.map((t): Structure[number] => {
        const sent = messageOf(t);
        return [
          t.key,
          groups?.of(t) ?? null,
          sent == null
            ? null
            : [
                sent.message,
                fieldsOf(formats, sent.icd, sent.message).map(treeFieldOf),
              ],
        ];
      }),
    );
  const structure = useSyncExternalStore(subscribe, getStructure, getStructure);
  const byKey = new Map<string, Node>();
  const groupNames = new Map<string, string>();
  /** The key of the group of each target in one. */
  const parents = new Map<string, string>();
  const nodes: Tree.Node[] = [];
  const groupNodes = new Map<string, Tree.Node[]>();
  for (const [target, group, sent] of JSON.parse(structure) as Structure) {
    byKey.set(target, { kind: "target", target });
    const node: Tree.Node =
      sent == null
        ? { key: target }
        : {
            key: target,
            children: sent[1].map((field) => {
              const key = fieldKey(target, sent[0], field.key);
              byKey.set(key, { kind: "field", target, field });
              return { key };
            }),
          };
    if (group == null) {
      nodes.push(node);
      continue;
    }
    const key = groupKey(group);
    parents.set(target, key);
    let children = groupNodes.get(key);
    if (children == null) {
      children = [];
      groupNodes.set(key, children);
      groupNames.set(key, group);
      nodes.push({ key, children });
    }
    children.push(node);
  }
  const tree = Tree.use({
    nodes,
    selected,
    onSelectedChange: onSelect,
    expanded,
    onExpandedChange: onExpand,
    toggleOn: "caret",
  });

  const targets = (): T[] => ctx.get(path).value;
  const disabledPath = (target: string) => deep.widen<Target[]>(path)[target].disabled;
  const sentBy = (target: string): TargetMessage | null => {
    const found = targets().find(({ key }) => key === target);
    if (found == null) throw new Error(`The task has no target ${target}`);
    return messageOf(found);
  };
  const sendsJSON = (target: string): boolean => {
    const sent = sentBy(target);
    if (sent == null) return false;
    return (
      formats
        .get(sent.icd)
        ?.value()
        .messages.find(({ key }) => key === sent.message)?.payload.format === "json"
    );
  };

  const addField = (target: string) => {
    const sent = sentBy(target);
    if (sent == null) throw new Error(`Target ${target} sends no message`);
    const fields = fieldsOf(formats, sent.icd, sent.message);
    const field = createJSONField(fields);
    formatOf(formats, sent.icd).insert(
      fieldsPath(sent.message),
      field,
      fields.at(-1)?.key,
    );
    setAdded(field.key);
    setLinked((prev) => [...prev, field.key]);
    const parent = parents.get(target);
    onExpand((prev) =>
      unique.unique([...prev, ...(parent == null ? [] : [parent]), target]),
    );
    onSelect([fieldKey(target, sent.message, field.key)]);
  };

  const handleSelect: UseTreeReturn["handleSelect"] = (keys, { clicked }) => {
    onSelect(keys);
    setAdded(undefined);
    if (clicked != null && byKey.get(clicked)?.kind !== "field") tree.expand(clicked);
  };

  /** @returns the chosen targets, with every target of each chosen group. */
  const chosenTargets = (keys: string[]): string[] => {
    const all = targets();
    return unique.unique(
      keys.flatMap((k) => {
        const name = groupNames.get(k);
        if (name != null && groups != null)
          return all.filter((t) => groups.of(t) === name).map(({ key }) => key);
        return byKey.get(k)?.kind === "target" ? [k] : [];
      }),
    );
  };
  const remove = (keys: string[]) => {
    PForm.fieldListUtils(ctx, path).remove(keys);
    onSelect([]);
  };
  const setEnabled = (keys: string[], enabled: boolean) =>
    ctx.batch(() => {
      for (const key of keys) ctx.set(disabledPath(key), !enabled);
    });
  const copy =
    duplicate == null
      ? undefined
      : (keys: string[]) => {
          const copies = targets()
            .filter(({ key }) => keys.includes(key))
            .map(duplicate);
          PForm.fieldListUtils(ctx, path).push(copies);
          onExpand((prev) => unique.unique([...prev, ...expandedOf(copies, groups)]));
          onSelect([copies[0].key]);
        };
  const contextMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    const chosen = chosenTargets(keys);
    const off = chosen.filter((key) => ctx.get(disabledPath(key)).value);
    const [only] = keys;
    const addable =
      keys.length === 1 && byKey.get(only)?.kind === "target" && sendsJSON(only);
    return (
      <Task.Views.ContextMenu
        keys={chosen}
        onRemove={remove}
        onEnable={off.length > 0 ? () => setEnabled(chosen, true) : undefined}
        onDisable={
          off.length < chosen.length ? () => setEnabled(chosen, false) : undefined
        }
        onAddField={addable ? () => addField(only) : undefined}
        onDuplicate={copy}
      />
    );
  };

  const renderItem = ({ key, ...p }: Tree.ItemRenderProps<string>) => {
    const name = groupNames.get(p.itemKey);
    if (name != null && groups != null)
      return <groups.Item key={key} {...p} name={name} />;
    const node = byKey.get(p.itemKey);
    if (node?.kind === "field") {
      const sent = sentBy(node.target);
      if (sent == null) throw new Error(`Target ${node.target} sends no message`);
      return (
        <FieldItem
          key={key}
          {...p}
          bindingsPath={sent.bindings}
          disabledPath={disabledPath(node.target)}
          field={node.field}
        />
      );
    }
    return <TargetItem key={key} {...p} />;
  };

  const [first] = selected;
  return {
    tree,
    handleSelect,
    contextMenu,
    renderItem,
    current: first == null ? undefined : byKey.get(first),
    currentGroup: first == null ? undefined : groupNames.get(first),
    addField,
    fieldProps: (field) => ({
      autoFocus: field === added,
      pointerLinked: linked.includes(field),
      onPointerChange: () => setLinked((prev) => prev.filter((k) => k !== field)),
    }),
  };
};
