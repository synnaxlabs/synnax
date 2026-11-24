// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import type { Tree } from "@synnaxlabs/lyra/tree";
import { compare, id, math, search, unique } from "@synnaxlabs/x";
import pluralize from "pluralize";
import { type ReactNode, useState } from "react";

import { Device } from "@/feature/mqtt/device";
import { pointerLabel, type Shape, shapesOf } from "@/feature/mqtt/task/entries";
import type { Leaf } from "@/feature/mqtt/task/leaves";
import { isSparkplugTopic } from "@/feature/mqtt/task/sparkplug";
import type { HeldEntry } from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { ICD } from "@/platform/icd";

const prettyPayload = (payload: string): string => {
  try {
    return JSON.stringify(JSON.parse(payload), null, 2);
  } catch {
    return payload;
  }
};

/** Turns "plant/line1" into the filter for everything under it. */
const toFilter = (scope: string): string => {
  const s = scope.trim().replace(/\/+$/, "");
  if (s === "") return "#";
  if (s.includes("#") || s.includes("+")) return s;
  return `${s}/#`;
};

/** A browsed topic with the shapes of its messages, parsed once per listen. */
interface ListedTopic extends Omit<Device.KeyedTopic, "shapes"> {
  shapes: Shape[];
}

const NO_TOPICS: ListedTopic[] = [];

interface Listing {
  /** Names the listen, so each one opens a fresh, fully expanded tree. */
  key: string;
  topics: ListedTopic[];
  truncated: boolean;
}

interface Folder {
  name: string;
  path: string;
  topic: ListedTopic | null;
  children: Folder[];
  /** Every topic at or under it. */
  topics: ListedTopic[];
}

const buildFolders = (topics: ListedTopic[]): Folder[] => {
  const root: Folder = { name: "", path: "", topic: null, children: [], topics: [] };
  for (const t of topics) {
    let node = root;
    const levels = t.topic.split("/");
    levels.forEach((name, i) => {
      const path = levels.slice(0, i + 1).join("/");
      let child = node.children.find((c) => c.path === path);
      if (child == null) {
        child = { name, path, topic: null, children: [], topics: [] };
        node.children.push(child);
      }
      node = child;
    });
    node.topic = t;
  }
  const finish = (nodes: Folder[]) => {
    nodes.sort(compare.by((n) => n.name));
    for (const n of nodes) {
      finish(n.children);
      n.topics = [
        ...(n.topic == null ? [] : [n.topic]),
        ...n.children.flatMap((c) => c.topics),
      ];
    }
  };
  finish(root.children);
  return root.children;
};

const byPath = (folders: Folder[], into = new Map<string, Folder>()) => {
  for (const f of folders) {
    into.set(f.path, f);
    byPath(f.children, into);
  }
  return into;
};

const toNodes = (folders: Folder[]): Tree.Node[] =>
  folders.map((f) => ({
    key: f.path,
    children: f.children.length === 0 ? undefined : toNodes(f.children),
  }));

/** Every level that holds deeper ones. */
const branchPaths = (topics: ListedTopic[]): string[] =>
  unique.unique(
    topics.flatMap(({ topic }) => {
      const levels = topic.split("/");
      return levels.slice(1).map((_, i) => levels.slice(0, i + 1).join("/"));
    }),
  );

/** The values picked from one shape of a topic. */
export interface ShapePick {
  shape: Shape;
  pointers: Set<string>;
  /** The pointer of the timestamp that indexes the picked values. */
  index?: string;
}

/** The values picked from one topic, by the key of their shape. */
export type TopicPick = Map<string, ShapePick>;

export type TopicPicks = Map<string, TopicPick>;

/** @returns the pick of the shape that the most messages had, which sets a format. */
export const formatOf = (pick: TopicPick): ShapePick | null =>
  ICD.Write.mostCommon([...pick.values()], ({ shape }) => shape.count ?? 0);

/** @returns the number of values picked from a topic. */
export const countOf = (pick: TopicPick): number =>
  math.sum([...pick.values()].map(({ pointers }) => pointers.size));

/** The pointers of each shape of a topic that the task takes, by shape key. */
type TopicTaken = Map<string, Set<string>>;

/** What the task takes of each topic that it holds. */
type Taken = Map<string, TopicTaken>;

/**
 * @returns what the task takes of each topic: of each shape, the fields of the message
 * that the Core gives its documents.
 */
const takenBy = (
  entries: HeldEntry[],
  topics: ListedTopic[],
  whole: boolean,
): Taken => {
  const taken: Taken = new Map();
  for (const { topic, shapes } of topics) {
    const entry = entries.find((e) => e.type === "plain" && e.topic === topic);
    if (entry?.type !== "plain") continue;
    const takenOf = (shape: Shape): { pointer: string }[] =>
      // A target holds a whole topic, so one in the task takes all its values.
      whole ? shape.leaves : (ICD.route(shape.leaves, entry.messages)?.fields ?? []);
    taken.set(
      topic,
      new Map(shapes.map((s) => [s.key, new Set(takenOf(s).map((f) => f.pointer))])),
    );
  }
  return taken;
};

/**
 * @returns the pointer of the timestamp that a pick of the shape takes with it. A topic
 * whose message in the task has fields keeps the index of its read.
 * @param taken - The pointers that the topic of the shape takes.
 */
const indexOf = ({ index }: Shape, taken: Set<string> | undefined, indexed: boolean) =>
  indexed && (taken?.size ?? 0) === 0 ? index : undefined;

/**
 * @returns the index that a pick takes only with other values. A topic whose only value
 * is its index, such as a heartbeat, can pick the index alone.
 */
const lockedIndex = (leaves: Leaf[], index?: string): string | undefined =>
  leaves.length > 1 ? index : undefined;

/** @returns the leaves of the shape that a pick can take. */
const freeLeaves = (
  shape: Shape,
  taken: Set<string> | undefined,
  indexed: boolean,
): Leaf[] => {
  const index = lockedIndex(shape.leaves, indexOf(shape, taken, indexed));
  return shape.leaves.filter(
    (l) => l.pointer !== index && taken?.has(l.pointer) !== true,
  );
};

/**
 * @returns a pick of each free leaf of each shape of the topic.
 * @param taken - What the task takes of the topic.
 */
const defaultPick = (
  { shapes }: ListedTopic,
  taken: TopicTaken | undefined,
  indexed: boolean,
): TopicPick => {
  const pick: TopicPick = new Map();
  for (const shape of shapes) {
    const own = taken?.get(shape.key);
    const pointers = new Set(freeLeaves(shape, own, indexed).map((l) => l.pointer));
    if (pointers.size > 0)
      pick.set(shape.key, { shape, pointers, index: indexOf(shape, own, indexed) });
  }
  return pick;
};

const toItem = (
  { name, path, topic, topics }: Folder,
  picks: TopicPicks,
  taken: Taken,
  whole: boolean,
  indexed: boolean,
): ICD.Picker.TreeItem => {
  const pickable = topics.filter(
    (t) => defaultPick(t, taken.get(t.topic), indexed).size > 0,
  );
  const all = pickable.length > 0 && pickable.every((t) => picks.has(t.topic));
  const values = whole
    ? topics.filter((t) => picks.has(t.topic)).length
    : math.sum(topics.map((t) => countOf(picks.get(t.topic) ?? new Map())));
  let meta = "";
  if (values > 0) meta = `${values} selected`;
  else if (topic != null && taken.has(topic.topic)) meta = "In task";
  else if (topic == null) meta = String(topics.length);
  let picked: ICD.Picker.TreeItem["picked"] = "none";
  if (all) picked = "all";
  else if (values > 0) picked = "some";
  return {
    key: path,
    name: name === "" ? "(empty level)" : name,
    meta,
    picked,
    disabled: pickable.length === 0,
    folder: topic == null,
  };
};

const toItems = (
  folders: Folder[],
  picks: TopicPicks,
  taken: Taken,
  whole: boolean,
  indexed: boolean,
  into: ICD.Picker.TreeItem[] = [],
): ICD.Picker.TreeItem[] => {
  for (const f of folders) {
    into.push(toItem(f, picks, taken, whole, indexed));
    toItems(f.children, picks, taken, whole, indexed, into);
  }
  return into;
};

/** @returns the title of the group of a shape, such as "44 messages with /temp". */
const titleOf = ({ count = 0, present }: Shape): string =>
  `${pluralize("message", count, true)}${present == null ? "" : ` with ${present}`}`;

interface TopicValuesProps {
  topic: ListedTopic;
  value: TopicPick;
  /** What the task takes of the topic. */
  taken: TopicTaken | undefined;
  indexed: boolean;
  whole: boolean;
  onChange: (value: TopicPick) => void;
}

const TopicValues = ({
  topic,
  value,
  taken,
  indexed,
  whole,
  onChange,
}: TopicValuesProps) => {
  const [raw, setRaw] = useState(false);
  const { shapes } = topic;
  // Shapes can share a pointer, so the key of a value holds the position of its shape.
  const keyOf = (shape: number, pointer: string) => `${shape}${pointer}`;
  const free = defaultPick(topic, taken, indexed);
  // A whole pick writes the fields of one shape alone.
  const format = whole && shapes.length > 1 ? formatOf(free)?.shape : undefined;
  const groups = shapes.map((shape, i): ICD.Picker.ValueGroup => {
    const own = taken?.get(shape.key);
    const locked = lockedIndex(shape.leaves, indexOf(shape, own, indexed));
    return {
      title: shapes.length > 1 ? titleOf(shape) : undefined,
      note: shape === format ? "Used for the format" : undefined,
      dimmed: format != null && shape !== format,
      values: shape.leaves.map((l): ICD.Picker.Value => {
        let lock: ICD.Picker.Lock | undefined;
        if (l.pointer === locked) lock = ICD.Picker.INDEX;
        else if (own?.has(l.pointer) === true) lock = ICD.Picker.TAKEN;
        return {
          key: keyOf(i, l.pointer),
          name: pointerLabel(l.pointer),
          sample: shape.samples.get(l.pointer) ?? null,
          dataType: ICD.DEFAULT_DATA_TYPES[l.kind],
          lock,
        };
      }),
    };
  });
  const picked = shapes.flatMap((shape, i) =>
    [...(value.get(shape.key)?.pointers ?? [])].map((p) => keyOf(i, p)),
  );
  const handleChange = (keys: string[]) => {
    const keyed = new Set(keys);
    const next: TopicPick = new Map();
    shapes.forEach((shape, i) => {
      const pointers = new Set(
        shape.leaves.map((l) => l.pointer).filter((p) => keyed.has(keyOf(i, p))),
      );
      const index = indexOf(shape, taken?.get(shape.key), indexed);
      if (pointers.size > 0) next.set(shape.key, { shape, pointers, index });
    });
    onChange(next);
  };
  return (
    <>
      <ICD.Picker.PaneHeader
        title={topic.topic}
        note={topic.retained ? "Retained" : undefined}
        free={countOf(free)}
        picked={picked.length}
        onPickAll={() => onChange(free)}
        onClear={() => onChange(new Map())}
      />
      <ICD.Picker.Values
        noun="Field"
        groups={groups}
        value={picked}
        // The tree picks a whole topic, so its fields are a preview.
        onChange={whole ? undefined : handleChange}
      />
      <Flex.Box y align="start" className={CSS.BE("mqtt-discover", "raw")}>
        <Button.Button variant="text" size="small" onClick={() => setRaw((r) => !r)}>
          {raw ? <Icon.Caret.Bottom /> : <Icon.Caret.Right />}
          Raw payload
        </Button.Button>
        {raw && (
          <pre className={CSS.BE("mqtt-discover", "raw-text")}>
            {prettyPayload(topic.payload)}
          </pre>
        )}
      </Flex.Box>
    </>
  );
};

const NOTHING_PUBLISHED =
  "Nothing published while Synnax listened. Add a rarely published topic by name.";

// Two folders of topics.
const SKELETON: ICD.Picker.SkeletonItem[] = [
  [0, "12ch"],
  [1, "17ch"],
  [1, "10ch"],
  [1, "14ch"],
  [0, "11ch"],
  [1, "16ch"],
  [1, "10ch"],
];

export interface TopicPickerProps {
  device: Device.Device;
  /** What the task holds now. */
  entries: HeldEntry[];
  /** Each pick is a whole topic, and a topic that the task holds takes none. */
  whole?: boolean;
  /** A pick of a topic that the task does not read takes its index too. */
  indexed?: boolean;
  value: TopicPicks;
  onChange: (value: TopicPicks) => void;
  onAddByName: (topic: string) => void;
  /** Shows at the foot of the list, such as a field that every pick shares. */
  children?: ReactNode;
}

/** Lists the JSON topics the broker publishes, and the fields of the one selected. */
export const TopicPicker = ({
  device,
  entries,
  whole = false,
  indexed = false,
  value,
  onChange,
  onAddByName,
  children,
}: TopicPickerProps) => {
  const [term, setTerm] = useState("");
  const [scope, setScope] = useState("");
  const [scopeDraft, setScopeDraft] = useState("");
  const [selected, setSelected] = useState<string | null>(null);
  const { result, listen } = ICD.Picker.useListen<Listing>(
    async (client) => {
      const r = await Device.browseTopics(client, device, toFilter(scope));
      const topics = r.topics
        // Sparkplug B payloads are not JSON. They have a tab of their own.
        .filter((t) => !isSparkplugTopic(t.topic))
        .map((t) => ({ ...t, shapes: shapesOf(t) }));
      return { key: id.create(), topics, truncated: r.truncated };
    },
    [device, scope],
  );
  const listing = result.data;
  const topics = listing?.topics ?? NO_TOPICS;

  const taken = takenBy(entries, topics, whole);
  const trimmed = term.trim();
  const visible = topics.filter((t) => search.matches(t.topic, trimmed));
  const folders = buildFolders(visible);
  const paths = byPath(folders);
  const nodes = toNodes(folders);
  const items = toItems(folders, value, taken, whole, indexed);
  const initialExpanded = branchPaths(topics);
  const active = selected ?? topics[0]?.topic ?? null;
  const activeTopic = topics.find((t) => t.topic === active) ?? null;

  const handlePick = (path: string) => {
    const folder = paths.get(path);
    if (folder == null) throw new Error(`No topic level at ${path}`);
    const pickable = folder.topics.filter(
      (t) => defaultPick(t, taken.get(t.topic), indexed).size > 0,
    );
    const all = pickable.length > 0 && pickable.every((t) => value.has(t.topic));
    const next = new Map(value);
    for (const t of pickable)
      if (all) next.delete(t.topic);
      else next.set(t.topic, defaultPick(t, taken.get(t.topic), indexed));
    onChange(next);
    if (folder.topic != null) setSelected(folder.path);
  };

  let tree: ReactNode;
  if (listing != null)
    tree = (
      <ICD.Picker.Tree
        key={listing.key}
        nodes={nodes}
        items={items}
        initialExpanded={initialExpanded}
        active={active}
        onActivate={setSelected}
        onPick={handlePick}
        emptyContent={
          <ICD.Picker.Notice
            message={
              trimmed === "" ? NOTHING_PUBLISHED : `No topic matches “${trimmed}”.`
            }
          />
        }
      />
    );
  else if (result.variant === "loading")
    tree = <ICD.Picker.TreeSkeleton items={SKELETON} />;
  else tree = <Status.Summary center status={result.status} />;

  const loading = result.variant === "loading";
  let statusText = pluralize("topic", topics.length, true);
  if (loading) statusText = Device.LISTENING;
  else if (listing?.truncated === true) statusText += ", list is full";

  let pane: ReactNode;
  if (activeTopic != null)
    pane = (
      <TopicValues
        key={activeTopic.topic}
        topic={activeTopic}
        value={value.get(activeTopic.topic) ?? new Map()}
        taken={taken.get(activeTopic.topic)}
        indexed={indexed}
        whole={whole}
        onChange={(pick) => {
          const next = new Map(value);
          if (pick.size === 0) next.delete(activeTopic.topic);
          else next.set(activeTopic.topic, pick);
          onChange(next);
        }}
      />
    );
  else if (listing == null && loading) pane = <ICD.Picker.PaneSkeleton noun="Field" />;
  else pane = <ICD.Picker.Notice message="Select a topic to see what it sends." />;

  return (
    <>
      <ICD.Picker.Status
        duration={Device.BROWSE_DURATION}
        status={statusText}
        loading={loading}
        onListen={listen}
      />
      <ICD.Picker.Side>
        <ICD.Picker.Search
          value={term}
          onChange={setTerm}
          placeholder="Search topics"
        />
        <ICD.Picker.SideList>{tree}</ICD.Picker.SideList>
        {(listing?.truncated === true || scope !== "") && (
          <Flex.Box y gap="small" className={CSS.BE("mqtt-discover", "scope")}>
            <Text.Text level="small" color={9}>
              Only list topics under
            </Text.Text>
            <Flex.Box x gap="small">
              <Input.Text
                value={scopeDraft}
                onChange={setScopeDraft}
                size="small"
                placeholder="plant/line1"
                className={CSS.BE("mqtt-discover", "code")}
              />
              <Button.Button
                variant="outlined"
                size="small"
                onClick={() => setScope(scopeDraft.trim())}
              >
                Listen
              </Button.Button>
            </Flex.Box>
          </Flex.Box>
        )}
        <ICD.Picker.SideAction onClick={() => onAddByName(trimmed)}>
          {trimmed === "" ? "Add a topic by name" : `Add “${trimmed}” as a topic`}
        </ICD.Picker.SideAction>
        {children}
      </ICD.Picker.Side>
      <ICD.Picker.Pane>{pane}</ICD.Picker.Pane>
    </>
  );
};
