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
import { deep, type record, unique, uuid } from "@synnaxlabs/x";
import {
  type Dispatch,
  type FC,
  type ReactElement,
  type ReactNode,
  type SetStateAction,
  useState,
} from "react";

import { copyName, copyPointer } from "@/platform/icd/fieldNames";
import { createJSONField } from "@/platform/icd/Fields";
import { fieldsOf, formatOf, useFormats } from "@/platform/icd/Formats";
import { fieldsPath, type TreeField, treeFieldOf } from "@/platform/icd/read";
import { Tree as ICDTree } from "@/platform/icd/tree";
import { FieldDetails } from "@/platform/icd/write/Details";
import { Task } from "@/platform/task";

/** A target of a write task, in the task's form. */
export interface Target extends record.Keyed<string> {
  disabled: boolean;
}

/** The message that a target sends. */
export interface TargetMessage {
  icd: icd.Key;
  message: icd.MessageKey;
  /** The path of the target's message in the task's form. */
  path: deep.Path<icd.WriteMessage>;
  /** @returns the name of the command channel that deploying creates for field. */
  channelName: (field: TreeField) => string;
}

/** The details of the selected item, as Task.Views.Panes shows them. */
export interface Pane {
  /** The form path of the item. */
  path: deep.Target<unknown>;
  title: ReactNode;
  details: ReactNode;
}

/** A node of the tree that is not a group. */
type Node =
  | { kind: "target"; target: string }
  | { kind: "field"; target: string; field: TreeField };

type FieldNode = Extract<Node, { kind: "field" }>;

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
  /** @returns the pane of a selected target. onAddField adds a field to its message. */
  targetPane: (target: T, onAddField: () => void) => Pane;
  /** @returns a copy of a target with a new key. Omitted, the menu has no Duplicate. */
  duplicate?: (target: T) => T;
  /** The key of each selected tree item. */
  selected: string[];
  onSelect: Dispatch<SetStateAction<string[]>>;
  /** What the tree shows open, as expandedOf and onExpand give it. */
  expanded: string[];
  onExpand: Dispatch<SetStateAction<string[]>>;
}

export interface UseTreeReturn {
  tree: Tree.UseReturn<string>;
  handleSelect: Tree.TreeProps<string, record.Keyed<string>>["onSelect"];
  contextMenu: (props: Menu.ContextMenuMenuProps) => ReactElement;
  renderItem: (props: Tree.ItemRenderProps<string>) => ReactElement;
  /**
   * The pane of the first selected node: a field opens its format and source, and a
   * target the pane that targetPane gives. Undefined for a group or no selection.
   */
  pane?: Pane;
  /** The group of the first selected node, or the node itself when it is a group. */
  currentGroup?: string;
}

const groupKey = (name: string): string => `group/${name}`;

/** Opens the group of a target, whatever the name of the group. */
const openKey = (target: string): string => `open/${target}`;

/** @returns the expanded state that shows the fields of each target. */
export const expandedOf = <T extends Target>(
  targets: T[],
  groups?: Groups<T>,
): string[] =>
  unique.unique(
    targets.flatMap((t) => (groups == null ? [t.key] : [openKey(t.key), t.key])),
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
  targetPane,
  duplicate,
  selected,
  onSelect,
  expanded,
  onExpand,
}: UseTreeArgs<T>): UseTreeReturn => {
  const ctx = PForm.useContext();
  const formats = useFormats();
  const all = PForm.useFieldValue(path);
  const [added, setAdded] = useState<icd.FieldKey>();
  const [created, setCreated] = useState<icd.FieldKey[]>([]);
  const structure = ICDTree.useStructure(formats, (): Structure =>
    ctx.get(path).value.map((t): Structure[number] => {
      const sent = messageOf(t);
      return [
        t.key,
        groups?.of(t) ?? null,
        sent == null
          ? null
          : [sent.message, fieldsOf(formats, sent.icd, sent.message).map(treeFieldOf)],
      ];
    }),
  );
  const byKey = new Map<string, Node>();
  const groupNames = new Map<string, string>();
  /** The open key of each target in a group. */
  const marks = new Set<string>();
  const nodes: Tree.Node[] = [];
  const groupNodes = new Map<string, Tree.Node[]>();
  for (const [target, group, sent] of structure) {
    byKey.set(target, { kind: "target", target });
    const node: Tree.Node =
      sent == null
        ? { key: target }
        : {
            key: target,
            children: sent[1].map((field) => {
              const key = ICDTree.fieldKey(target, sent[0], field.key);
              byKey.set(key, { kind: "field", target, field });
              return { key };
            }),
          };
    if (group == null) {
      nodes.push(node);
      continue;
    }
    const key = groupKey(group);
    marks.add(openKey(target));
    let children = groupNodes.get(key);
    if (children == null) {
      children = [];
      groupNodes.set(key, children);
      groupNames.set(key, group);
      nodes.push({ key, children });
    }
    children.push(node);
  }
  // A group is open while one of its targets is, so a target that moves to another
  // group stays in view.
  const open = new Set(expanded);
  const shown = [
    ...expanded.filter((k) => !marks.has(k)),
    ...[...groupNodes]
      .filter(([, children]) => children.some(({ key }) => open.has(openKey(key))))
      .map(([key]) => key),
  ];
  const handleExpand = (arg: SetStateAction<string[]>) => {
    const next = typeof arg === "function" ? arg(shown) : arg;
    onExpand([
      ...next.filter((k) => !marks.has(k) && !groupNodes.has(k)),
      ...[...groupNodes]
        .filter(([key]) => next.includes(key))
        .flatMap(([, children]) => children.map(({ key }) => openKey(key))),
    ]);
  };
  const tree = Tree.use({
    nodes,
    selected,
    onSelectedChange: onSelect,
    expanded: shown,
    onExpandedChange: handleExpand,
    toggleOn: "caret",
  });

  const targets = (): T[] => ctx.get(path).value;
  const disabledPath = (target: string) => deep.widen<Target[]>(path)[target].disabled;
  const sentBy = (target: string): TargetMessage | null => {
    const found = targets().find(({ key }) => key === target);
    if (found == null) throw new Error(`The task has no target ${target}`);
    return messageOf(found);
  };
  const sending = (target: string): TargetMessage => {
    const sent = sentBy(target);
    if (sent == null) throw new Error(`Target ${target} sends no message`);
    return sent;
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

  /**
   * Adds a field to the JSON message of a target and to its ICD, and selects it. The
   * Suit of the target binds it.
   */
  const addField = (target: string) => {
    const sent = sending(target);
    const fields = fieldsOf(formats, sent.icd, sent.message);
    const field = createJSONField(fields);
    formatOf(formats, sent.icd).insert(
      fieldsPath(sent.message),
      field,
      fields.at(-1)?.key,
    );
    setAdded(field.key);
    setCreated((prev) => [...prev, field.key]);
    const mark = openKey(target);
    onExpand((prev) =>
      unique.unique([...prev, ...(marks.has(mark) ? [mark] : []), target]),
    );
    onSelect([ICDTree.fieldKey(target, sent.message, field.key)]);
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
  const removeTargets = (keys: string[]) => {
    PForm.fieldListUtils(ctx, path).remove(keys);
    onSelect([]);
  };
  const setEnabled = (keys: string[], enabled: boolean) =>
    ctx.batch(() => {
      for (const key of keys) ctx.set(disabledPath(key), !enabled);
    });
  const duplicateTargets =
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
  /**
   * Removes each chosen field from its message in the ICD and from each target that
   * sends the message.
   */
  const removeFields = (chosen: FieldNode[]) => {
    ctx.batch(() => {
      for (const { target, field } of chosen) {
        const { icd: icdKey, message } = sending(target);
        formatOf(formats, icdKey).remove(fieldsPath(message)[field.key]);
        for (const t of targets()) {
          const sent = messageOf(t);
          if (sent?.message !== message) continue;
          const bindings = ctx.get(sent.path.bindings).value;
          ctx.set(
            sent.path.bindings,
            bindings.filter((b) => b.field !== field.key),
          );
        }
      }
    });
    onSelect([]);
  };

  /**
   * Copies each chosen field after it in its message, at a pointer of its own. Each
   * chosen target that binds the field binds its copy the same way, with a new channel
   * for a channel. Selects the first copy.
   */
  const duplicateFields = (chosen: FieldNode[]) => {
    const copies = new Map<string, icd.JSONField>();
    const keys: string[] = [];
    ctx.batch(() => {
      for (const { target, field } of chosen) {
        const sent = sending(target);
        const fields = fieldsOf(formats, sent.icd, sent.message);
        const at = `${sent.message}/${field.key}`;
        let copy = copies.get(at);
        if (copy == null) {
          const full = fields.find(({ key }) => key === field.key);
          if (full == null)
            throw new Error(`Message ${sent.message} has no field ${field.key}`);
          // A write sends each field, so the copy cannot share the pointer.
          copy = {
            ...full,
            key: uuid.create(),
            name: copyName(
              field.name,
              fields.map(({ name }) => name),
            ),
            pointer: copyPointer(
              full.pointer,
              fields.map(({ pointer }) => pointer),
            ),
          };
          formatOf(formats, sent.icd).insert(fieldsPath(sent.message), copy, field.key);
          copies.set(at, copy);
        }
        const bindings = ctx.get(sent.path.bindings).value;
        const binding = bindings.find((b) => b.field === field.key);
        if (binding != null)
          ctx.set(sent.path.bindings, [
            ...bindings,
            binding.type === "channel"
              ? { ...binding, field: copy.key, channel: 0 }
              : { ...binding, field: copy.key },
          ]);
        keys.push(ICDTree.fieldKey(target, sent.message, copy.key));
      }
    });
    onSelect(keys.slice(0, 1));
  };

  const contextMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    const fields = keys.flatMap((k) => {
      const node = byKey.get(k);
      return node?.kind === "field" ? [node] : [];
    });
    if (fields.length > 0) {
      const onlyFields = fields.length === keys.length;
      // A field at the whole document leaves no pointer free for a copy.
      const copyable = onlyFields && fields.every(({ field }) => field.pointer !== "");
      return (
        <Task.Views.ContextMenu
          keys={keys}
          onRemove={onlyFields ? () => removeFields(fields) : undefined}
          onDuplicate={copyable ? () => duplicateFields(fields) : undefined}
        />
      );
    }
    const chosen = chosenTargets(keys);
    const off = chosen.filter((key) => ctx.get(disabledPath(key)).value);
    const [only] = keys;
    const addable =
      keys.length === 1 && byKey.get(only)?.kind === "target" && sendsJSON(only);
    return (
      <Task.Views.ContextMenu
        keys={chosen}
        onRemove={removeTargets}
        onEnable={off.length > 0 ? () => setEnabled(chosen, true) : undefined}
        onDisable={
          off.length < chosen.length ? () => setEnabled(chosen, false) : undefined
        }
        onAddField={addable ? () => addField(only) : undefined}
        onDuplicate={duplicateTargets}
      />
    );
  };

  const renderItem = ({ key, ...p }: Tree.ItemRenderProps<string>) => {
    const name = groupNames.get(p.itemKey);
    if (name != null && groups != null)
      return <groups.Item key={key} {...p} name={name} />;
    const node = byKey.get(p.itemKey);
    if (node?.kind === "field") {
      const sent = sending(node.target);
      return (
        <FieldItem
          key={key}
          {...p}
          bindingsPath={sent.path.bindings}
          disabledPath={disabledPath(node.target)}
          field={node.field}
        />
      );
    }
    return <TargetItem key={key} {...p} />;
  };

  const [first] = selected;
  const paneOf = (node: Node, target: T): Pane => {
    if (node.kind === "target") return targetPane(target, () => addField(target.key));
    const sent = sending(node.target);
    const i = ctx
      .get(sent.path.bindings)
      .value.findIndex(({ field }) => field === node.field.key);
    return {
      path: i < 0 ? sent.path : sent.path.bindings[i],
      title: (
        <Text.Text level="p" weight={500} overflow="fade">
          {node.field.name}
        </Text.Text>
      ),
      details: (
        <FieldDetails
          key={first}
          path={sent.path.bindings}
          icdKey={sent.icd}
          message={sent.message}
          field={node.field.key}
          channelName={sent.channelName(node.field)}
          skipped={target.disabled}
          autoFocus={node.field.key === added}
          created={created.includes(node.field.key)}
        />
      ),
    };
  };
  const node = first == null ? undefined : byKey.get(first);
  const target = node == null ? undefined : all.find(({ key }) => key === node.target);
  if (node != null && target == null)
    throw new Error(`The task has no target ${node.target}`);
  return {
    tree,
    handleSelect,
    contextMenu,
    renderItem,
    pane: node == null || target == null ? undefined : paneOf(node, target),
    currentGroup:
      (target == null ? undefined : groups?.of(target)) ??
      (first == null ? undefined : groupNames.get(first)),
  };
};
