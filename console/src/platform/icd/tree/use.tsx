// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Tree } from "@synnaxlabs/lyra/tree";
import { ICD as PICD } from "@synnaxlabs/pluto";
import { deep, type record, unique, uuid } from "@synnaxlabs/x";
import {
  type Dispatch,
  type FC,
  type ReactElement,
  type SetStateAction,
  useSyncExternalStore,
} from "react";

import { MessageLabel, ReadCheckbox } from "@/platform/icd/Bindings";
import { createJSONField } from "@/platform/icd/Fields";
import { fieldsOf, formatOf, type Formats, useFormats } from "@/platform/icd/Formats";
import { createMatch } from "@/platform/icd/Match";
import { usePlan } from "@/platform/icd/Plans";
import { fieldsPath, isReadable, type ReadSource, reindex } from "@/platform/icd/read";
import {
  fieldKey,
  type FieldNode,
  messageKey,
  type MessageNode,
  type Node,
  type SourceNode,
} from "@/platform/icd/tree/nodes";
import { Task } from "@/platform/task";

/** A source of a read task, such as a topic or a record, in the task's form. */
export interface Source extends record.Keyed<string> {
  disabled: boolean;
}

/** Starts or stops reading a field of a message. */
const setRead = (
  ctx: PForm.ContextValue,
  path: deep.Path<ReadSource>,
  read: number,
  field: icd.JSONField,
  on: boolean,
  bindingOf: () => icd.ReadBinding,
): void => {
  const bindings = path.messages[read].bindings;
  const value = ctx.get(bindings).value;
  const bound = value.some(({ field: key }) => key === field.key);
  if (bound === on || (on && !isReadable(field))) return;
  ctx.set(
    bindings,
    on ? [...value, bindingOf()] : value.filter(({ field: k }) => k !== field.key),
  );
};

interface Index {
  nodes: Tree.Node[];
  byKey: Map<string, Node>;
  /** The key of each group. */
  groups: Set<string>;
}

/** A list of sources of the task, and the key of its group, or null with no groups. */
type List<S extends Source> = [string | null, deep.Target<S[]>];

/** Each source key with the message and shown fields of each read, or null for none. */
type Sources = [string, [icd.MessageKey, icd.JSONField[]][] | null][];

/** The sources of each list, by the key of its group. */
type Structure = [string | null, Sources][];

/**
 * The sources of the task as tree nodes, under the node of their group when they have
 * one. A source holds the fields of its message, or a node for each message when it
 * reads several. A source that reads no messages stands alone. The index field stays
 * hidden, as the timestamp section edits it.
 */
const useIndex = <S extends Source>(
  lists: () => List<S>[],
  sourceOf: (source: S) => ReadSource | null,
  formats: Formats,
): Index => {
  const { bind, get } = PForm.useContext();
  const subscribe = (onChange: () => void) => {
    const unbinds = [
      bind(onChange),
      ...[...formats.values()].map((f) => f.bind(onChange)),
    ];
    return () => unbinds.forEach((unbind) => unbind());
  };
  const getStructure = () =>
    JSON.stringify(
      lists().map(([group, sources]): Structure[number] => [
        group,
        get(sources).value.map((s): Sources[number] => {
          const source = sourceOf(s);
          return [
            s.key,
            source == null
              ? null
              : source.messages.map(({ message, index }) => [
                  message,
                  fieldsOf(formats, source.icd, message).filter(
                    ({ key }) => key !== index,
                  ),
                ]),
          ];
        }),
      ]),
    );
  const structure = useSyncExternalStore(subscribe, getStructure, getStructure);
  const byKey = new Map<string, Node>();
  const groups = new Set<string>();
  const sourceNode = ([source, reads]: Sources[number]): Tree.Node => {
    byKey.set(source, { kind: "source", source });
    if (reads == null) return { key: source };
    const fieldNodes = (
      read: number,
      message: icd.MessageKey,
      fields: icd.JSONField[],
    ): Tree.Node[] =>
      fields.map((field) => {
        const key = fieldKey(source, message, field.key);
        byKey.set(key, { kind: "field", source, read, field });
        return { key };
      });
    if (reads.length === 1)
      return { key: source, children: fieldNodes(0, ...reads[0]) };
    const children = reads.map(([message, fields], read): Tree.Node => {
      const key = messageKey(source, message);
      byKey.set(key, { kind: "message", source, read });
      return { key, children: fieldNodes(read, message, fields) };
    });
    return { key: source, children };
  };
  const nodes = (JSON.parse(structure) as Structure).flatMap(
    ([group, sources]): Tree.Node[] => {
      const children = sources.map(sourceNode);
      if (group == null) return children;
      groups.add(group);
      return [{ key: group, children }];
    },
  );
  return { nodes, byKey, groups };
};

interface MessageItemProps extends Tree.ItemRenderProps<string> {
  path: deep.Path<ReadSource>;
  read: number;
}

const MessageItem = ({ path, read, ...props }: MessageItemProps) => (
  <Task.Views.TreeGroupItem
    {...props}
    label={<MessageLabel path={path} read={read} />}
  />
);

interface FieldItemProps extends Tree.ItemRenderProps<string> {
  path: deep.Path<ReadSource>;
  node: FieldNode;
  onChange: (on: boolean) => void;
}

const FieldItem = ({
  path,
  node: { read, field },
  onChange,
  ...props
}: FieldItemProps) => {
  const bindingsPath = path.messages[read].bindings;
  const bindings = PForm.useFieldValue(bindingsPath);
  const sourceOff = PForm.useFieldValue(path.disabled);
  const plan = usePlan(path, read);
  const i = bindings.findIndex(({ field: key }) => key === field.key);
  const replaced =
    i >= 0 &&
    bindings[i].channel !== 0 &&
    plan != null &&
    Task.newChannelName(plan.channels, field.key) != null;
  return (
    <Task.Views.TreeFieldItem
      {...props}
      label={field.name}
      emptyLabel="Unnamed field"
      channel={i < 0 ? null : bindingsPath[i].channel}
      replaced={replaced}
      off={i < 0}
      skipped={i < 0 || sourceOff}
    >
      <ReadCheckbox bindings={bindingsPath} field={field} onChange={onChange} />
    </Task.Views.TreeFieldItem>
  );
};

/** The groups of the sources of a task, such as topics that hold records. */
interface Groups<G extends record.Keyed<string>, S extends Source> {
  /** The groups of the task in its form. */
  path: deep.Target<G[]>;
  /** @returns the path of the sources of a group. */
  sourcesOf: (group: string) => deep.Target<S[]>;
  /** Renders the tree item of a group. */
  Item: FC<Tree.ItemRenderProps<string>>;
}

/** Where the sources of a task are in its form: in one list, or in groups. */
type Layout<G extends record.Keyed<string>, S extends Source> =
  | { sources: deep.Target<S[]>; groups?: never }
  | { groups: Groups<G, S>; sources?: never };

interface BaseArgs<S extends Source> {
  /** @returns the source as one that reads ICD messages, or null when it reads none. */
  sourceOf: (source: S) => ReadSource | null;
  /** @returns the path of a source that reads ICD messages. */
  pathOf: (key: string) => deep.Path<ReadSource>;
  /** Renders the tree item of a source. */
  SourceItem: FC<Tree.ItemRenderProps<string>>;
  /** @returns the binding of a field that the task starts to read. */
  bindingOf: (source: string, r: number, field: icd.JSONField) => icd.ReadBinding;
  /** @returns a name for a new message of a source that no message in held has. */
  messageName: (source: string, held: icd.Message[]) => string;
  /** The key of each selected tree item. */
  selected: string[];
  onSelect: Dispatch<SetStateAction<string[]>>;
  expanded: string[];
  onExpand: Dispatch<SetStateAction<string[]>>;
}

export type UseArgs<
  S extends Source,
  G extends record.Keyed<string> = record.Keyed<string>,
> = BaseArgs<S> & Layout<G, S>;

export interface UseReturn {
  tree: Tree.UseReturn<string>;
  handleSelect: Tree.TreeProps<string, record.Keyed<string>>["onSelect"];
  contextMenu: (props: Menu.ContextMenuMenuProps) => ReactElement;
  renderItem: (props: Tree.ItemRenderProps<string>) => ReactElement;
  /** The first selected node, unless it is a group. */
  current?: Node;
  /** The key of the first selected node, when it is a group. */
  currentGroup?: string;
  /** Adds a field to a message of a source and to its ICD, reads it, and selects it. */
  addField: (source: string, read: number) => void;
  /**
   * Adds a message with no fields to a source and to its ICD. The match tests the
   * pointer of the source's other matches, so the messages tell payloads apart.
   */
  addMessage: (source: string) => void;
}

/**
 * The tree of the sources of a read task, with each message and field that they read.
 * It edits the ICDs of the sources through their open forms, so call it inside a
 * {@link WithFormat} for the ICD of each source. Source keys must be unique across
 * groups.
 */
export const use = <
  S extends Source,
  G extends record.Keyed<string> = record.Keyed<string>,
>({
  sourceOf,
  pathOf,
  SourceItem,
  bindingOf,
  messageName,
  selected,
  onSelect,
  expanded,
  onExpand,
  ...layout
}: UseArgs<S, G>): UseReturn => {
  const ctx = PForm.useContext();
  const formats = useFormats();
  const { groups: grouping } = layout;
  const lists = (): List<S>[] =>
    grouping == null
      ? [[null, layout.sources]]
      : ctx
          .get(grouping.path)
          .value.map(({ key }): List<S> => [key, grouping.sourcesOf(key)]);
  const all = (): S[] => lists().flatMap(([, list]) => ctx.get(list).value);
  /** @returns the list that holds a source. */
  const listOf = (source: string): deep.Target<S[]> => {
    const found = lists().find(([, list]) =>
      ctx.get(list).value.some(({ key }) => key === source),
    );
    if (found == null) throw new Error(`No list of the task holds source ${source}`);
    return found[1];
  };
  const { nodes, byKey, groups } = useIndex(lists, sourceOf, formats);
  const tree = Tree.use({
    nodes,
    selected,
    onSelectedChange: onSelect,
    expanded,
    onExpandedChange: onExpand,
    toggleOn: "caret",
  });
  const disabledPath = (source: string) =>
    deep.widen<Source[]>(listOf(source))[source].disabled;
  const read = (source: string, r: number, field: icd.JSONField, on: boolean) =>
    setRead(ctx, pathOf(source), r, field, on, () => bindingOf(source, r, field));

  const addMessage = (sourceKey: string) => {
    const path = pathOf(sourceKey);
    const source = ctx.get(path).value;
    const form = formatOf(formats, source.icd);
    const held = form.value().messages;
    const rule = source.messages
      .map(({ message }) => held.find(({ key }) => key === message)?.payload)
      .find((p) => p?.format === "json" && p.match != null);
    const match = rule?.format === "json" ? rule.match : undefined;
    const message: icd.Message = {
      key: uuid.create(),
      name: messageName(sourceKey, held),
      payload: {
        format: "json",
        match: createMatch(match?.type ?? "string", match?.pointer ?? ""),
        fields: [],
      },
    };
    form.insert(PICD.formPath.messages, message, held.at(-1)?.key);
    ctx.set(path.messages, [
      ...source.messages,
      { message: message.key, bindings: [] },
    ]);
    const shown = [...source.messages.map((m) => m.message), message.key];
    onExpand((prev) =>
      unique.unique([
        ...prev,
        sourceKey,
        ...shown.map((m) => messageKey(sourceKey, m)),
      ]),
    );
    onSelect([messageKey(sourceKey, message.key)]);
  };

  /** Whether the ICD holds the message of a read of a source, as JSON. */
  const holdsJSON = (sourceKey: string, r: number): boolean => {
    const { icd: icdKey, messages } = ctx.get(pathOf(sourceKey)).value;
    const message = messages[r].message;
    return (
      formats
        .get(icdKey)
        ?.value()
        .messages.find(({ key }) => key === message)?.payload.format === "json"
    );
  };

  const addField = (sourceKey: string, r: number) => {
    const source = ctx.get(pathOf(sourceKey)).value;
    const { message } = source.messages[r];
    const fields = fieldsOf(formats, source.icd, message);
    const field = createJSONField(fields.map(({ name }) => name));
    formatOf(formats, source.icd).insert(
      fieldsPath(message),
      field,
      fields.at(-1)?.key,
    );
    read(sourceKey, r, field, true);
    const parent = source.messages.length > 1 ? [messageKey(sourceKey, message)] : [];
    onExpand((prev) => unique.unique([...prev, sourceKey, ...parent]));
    onSelect([fieldKey(sourceKey, message, field.key)]);
  };

  /**
   * Removes each chosen field from its message in the ICD and from each read of it. A
   * read whose index is the field gets no index.
   */
  const removeFields = (chosen: FieldNode[]) => {
    ctx.batch(() => {
      for (const { source, read: r, field } of chosen) {
        const { icd: icdKey, messages } = ctx.get(pathOf(source)).value;
        const { message } = messages[r];
        formatOf(formats, icdKey).remove(fieldsPath(message)[field.key]);
        for (const s of all())
          sourceOf(s)?.messages.forEach((m, i) => {
            if (m.message !== message) return;
            if (m.index === field.key)
              ctx.set(pathOf(s.key).messages[i], reindex(m, undefined, 0));
            else read(s.key, i, field, false);
          });
      }
    });
    onSelect([]);
  };

  const handleSelect: UseReturn["handleSelect"] = (keys, { clicked }) => {
    onSelect(keys);
    const kind = clicked == null ? undefined : byKey.get(clicked)?.kind;
    if (clicked != null && kind !== "field") tree.expand(clicked);
  };

  /** Sets whether the task reads each chosen field, and enables each chosen source. */
  const setEnabled = (chosen: (SourceNode | FieldNode)[], enabled: boolean) =>
    ctx.batch(() => {
      for (const node of chosen)
        if (node.kind === "source") ctx.set(disabledPath(node.source), !enabled);
        else read(node.source, node.read, node.field, enabled);
    });
  const isEnabled = (node: SourceNode | FieldNode): boolean => {
    if (node.kind === "source") return !ctx.get(disabledPath(node.source)).value;
    return ctx
      .get(pathOf(node.source).messages[node.read].bindings)
      .value.some(({ field }) => field === node.field.key);
  };
  /** Removes each chosen source, and each chosen message from its source. */
  const removeNodes = (chosen: (SourceNode | MessageNode)[]) => {
    const removed = new Set(
      chosen.flatMap((n) => (n.kind === "source" ? [n.source] : [])),
    );
    const dropped = new Map<string, Set<number>>();
    for (const n of chosen)
      if (n.kind === "message" && !removed.has(n.source))
        dropped.set(n.source, (dropped.get(n.source) ?? new Set()).add(n.read));
    ctx.batch(() => {
      dropped.forEach((reads, source) => {
        const path = pathOf(source).messages;
        ctx.set(
          path,
          ctx.get(path).value.filter((_, r) => !reads.has(r)),
        );
      });
      if (removed.size > 0)
        lists().forEach(([, list]) =>
          PForm.fieldListUtils(ctx, list).remove([...removed]),
        );
    });
    onSelect([]);
  };
  /** Removes each chosen group with its sources. */
  const removeGroups = (chosen: string[]) => {
    if (grouping == null) throw new Error("The task tree has no groups to remove");
    PForm.fieldListUtils(ctx, grouping.path).remove(chosen);
    onSelect([]);
  };
  /**
   * @returns whether the chosen nodes can go: sources, and messages that leave each of
   * their sources at least one.
   */
  const isRemovable = (chosen: Node[]): chosen is (SourceNode | MessageNode)[] => {
    if (chosen.some(({ kind }) => kind === "field")) return false;
    const counts = new Map<string, number>();
    for (const n of chosen)
      if (n.kind === "message") counts.set(n.source, (counts.get(n.source) ?? 0) + 1);
    return [...counts].every(
      ([source, count]) =>
        chosen.some((n) => n.kind === "source" && n.source === source) ||
        count < ctx.get(pathOf(source).messages).value.length,
    );
  };
  /** @returns whether the source of a key reads ICD messages. */
  const readsMessages = (key: string): boolean =>
    all().some((s) => s.key === key && sourceOf(s) != null);
  const contextMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    const chosenGroups = keys.filter((k) => groups.has(k));
    if (chosenGroups.length > 0)
      return (
        <Task.Views.ContextMenu
          keys={keys}
          onRemove={
            chosenGroups.length === keys.length
              ? () => removeGroups(chosenGroups)
              : undefined
          }
        />
      );
    const chosen = keys.flatMap((k) => byKey.get(k) ?? []);
    const switchable = chosen.filter(
      (n): n is SourceNode | FieldNode => n.kind !== "message",
    );
    const [only] = chosen;
    const source =
      chosen.length === 1 && only.kind === "source" && readsMessages(only.source)
        ? only.source
        : undefined;
    let onAddField: (() => void) | undefined;
    if (
      chosen.length === 1 &&
      only.kind === "message" &&
      holdsJSON(only.source, only.read)
    )
      onAddField = () => addField(only.source, only.read);
    else if (
      source != null &&
      ctx.get(pathOf(source).messages).value.length === 1 &&
      holdsJSON(source, 0)
    )
      onAddField = () => addField(source, 0);
    const fields = chosen.filter((n): n is FieldNode => n.kind === "field");
    let onRemove: (() => void) | undefined;
    if (fields.length > 0 && fields.length === chosen.length)
      onRemove = () => removeFields(fields);
    else if (isRemovable(chosen)) onRemove = () => removeNodes(chosen);
    return (
      <Task.Views.ContextMenu
        keys={keys}
        onRemove={onRemove}
        onEnable={
          switchable.some((n) => !isEnabled(n))
            ? () => setEnabled(switchable, true)
            : undefined
        }
        onDisable={
          switchable.some(isEnabled) ? () => setEnabled(switchable, false) : undefined
        }
        onAddField={onAddField}
        extraMenuItems={
          source != null && (
            <Menu.Item itemKey="addMessage" onClick={() => addMessage(source)}>
              <Icon.Add />
              Add message
            </Menu.Item>
          )
        }
      />
    );
  };

  const renderItem = ({ key, ...p }: Tree.ItemRenderProps<string>) => {
    if (grouping != null && groups.has(p.itemKey))
      return <grouping.Item key={key} {...p} />;
    const node = byKey.get(p.itemKey);
    if (node?.kind === "field")
      return (
        <FieldItem
          key={key}
          {...p}
          path={pathOf(node.source)}
          node={node}
          onChange={(on) => read(node.source, node.read, node.field, on)}
        />
      );
    if (node?.kind === "message")
      return (
        <MessageItem key={key} {...p} path={pathOf(node.source)} read={node.read} />
      );
    return <SourceItem key={key} {...p} />;
  };

  const [first] = selected;
  const current = first == null ? undefined : byKey.get(first);
  const currentGroup = first != null && groups.has(first) ? first : undefined;
  return {
    tree,
    handleSelect,
    contextMenu,
    renderItem,
    current,
    currentGroup,
    addField,
    addMessage,
  };
};
