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
import { type Tree } from "@synnaxlabs/lyra/tree";
import { id, unique } from "@synnaxlabs/x";
import pluralize from "pluralize";
import { type ReactNode, useState } from "react";

import { browseTopics, type KeyedTopic } from "@/feature/mqtt/device/browse";
import { type Device } from "@/feature/mqtt/device/types";
import { Picker } from "@/feature/mqtt/task/discover/picker";
import {
  PaneHeader,
  PaneSkeleton,
  type Value,
  Values,
} from "@/feature/mqtt/task/discover/Values";
import { leavesOf, pointerLabel, sampleAt } from "@/feature/mqtt/task/entries";
import { isSparkplugTopic } from "@/feature/mqtt/task/sparkplug";
import {
  type ReadEntry,
  type ReadField,
  type WriteTarget,
} from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";

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

/** A browsed topic with the fields its payload offers, parsed once per listen. */
interface ListedTopic extends KeyedTopic {
  leaves: ReadField[];
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
    nodes.sort((a, b) => a.name.localeCompare(b.name));
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

/**
 * The values a click on a topic picks: its numbers and booleans, or all of them. A
 * single pick takes the first.
 */
const defaultPointers = (
  leaves: ReadField[],
  taken: Set<string> | undefined,
  single: boolean,
): Set<string> => {
  const free = leaves.filter((l) => taken?.has(l.pointer) !== true);
  const scalars = free.filter((l) => !l.disabled);
  const picked = scalars.length > 0 ? scalars : free;
  return new Set((single ? picked.slice(0, 1) : picked).map((l) => l.pointer));
};

/** The values picked from one topic, with the fields they came from. */
export interface TopicPick {
  leaves: ReadField[];
  pointers: Set<string>;
}

export type TopicPicks = Map<string, TopicPick>;

/** The pointers that each topic of the task takes. */
type Taken = Map<string, Set<string>>;

const takenBy = (
  entries: (ReadEntry | WriteTarget)[],
  topics: ListedTopic[],
  single: boolean,
): Taken => {
  const plain = entries.flatMap((e) => (e.type === "plain" ? [e] : []));
  if (!single)
    return new Map(
      plain.map((e) => [e.topic, new Set(e.fields.map((f) => f.pointer))]),
    );
  // A target holds a whole topic, so one in the task takes all its values.
  const held = new Set(plain.map((e) => e.topic));
  return new Map(
    topics
      .filter((t) => held.has(t.topic))
      .map((t) => [t.topic, new Set(t.leaves.map((l) => l.pointer))]),
  );
};

const freeCount = (t: ListedTopic, taken: Taken) =>
  t.leaves.filter((l) => taken.get(t.topic)?.has(l.pointer) !== true).length;

const toItem = (
  { name, path, topic, topics }: Folder,
  picks: TopicPicks,
  taken: Taken,
): Picker.TreeItem => {
  const pickable = topics.filter((t) => freeCount(t, taken) > 0);
  const all = pickable.length > 0 && pickable.every((t) => picks.has(t.topic));
  const values = topics.reduce(
    (sum, t) => sum + (picks.get(t.topic)?.pointers.size ?? 0),
    0,
  );
  let meta = "";
  if (values > 0) meta = `${values} selected`;
  else if (topic != null && taken.has(topic.topic)) meta = "In task";
  else if (topic == null) meta = String(topics.length);
  let picked: Picker.TreeItem["picked"] = "none";
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
  into: Picker.TreeItem[] = [],
): Picker.TreeItem[] => {
  for (const f of folders) {
    into.push(toItem(f, picks, taken));
    toItems(f.children, picks, taken, into);
  }
  return into;
};

interface TopicValuesProps {
  topic: ListedTopic;
  value: Set<string>;
  taken: Set<string>;
  single: boolean;
  onChange: (value: Set<string>) => void;
}

const TopicValues = ({ topic, value, taken, single, onChange }: TopicValuesProps) => {
  const [raw, setRaw] = useState(false);
  const { leaves } = topic;
  const free = leaves.filter((l) => !taken.has(l.pointer)).map((l) => l.pointer);
  const values: Value[] = leaves.map((l) => ({
    key: l.pointer,
    name: pointerLabel(l.pointer),
    sample: sampleAt(topic, l.pointer),
    dataType: l.dataType,
    taken: taken.has(l.pointer),
  }));
  return (
    <>
      <PaneHeader
        title={topic.topic}
        note={topic.retained ? "Retained" : undefined}
        free={free.length}
        picked={value.size}
        onPickAll={single ? undefined : () => onChange(new Set(free))}
        onClear={() => onChange(new Set())}
      />
      <Values
        noun="Field"
        groups={[{ values }]}
        value={[...value]}
        // A single pick replaces the one before.
        onChange={(keys) =>
          onChange(new Set(single ? keys.filter((k) => !value.has(k)).slice(-1) : keys))
        }
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
const SKELETON: Picker.SkeletonItem[] = [
  [0, "12ch"],
  [1, "17ch"],
  [1, "10ch"],
  [1, "14ch"],
  [0, "11ch"],
  [1, "16ch"],
  [1, "10ch"],
];

export interface TopicPickerProps {
  device: Device;
  /** What the task holds now. */
  entries: (ReadEntry | WriteTarget)[];
  /** Each topic takes one value, and a topic that the task holds takes none. */
  single?: boolean;
  value: TopicPicks;
  onChange: (value: TopicPicks) => void;
  onAddByName: (topic: string) => void;
}

/** Lists the JSON topics the broker publishes, and the fields of the one selected. */
export const TopicPicker = ({
  device,
  entries,
  single = false,
  value,
  onChange,
  onAddByName,
}: TopicPickerProps) => {
  const [search, setSearch] = useState("");
  const [scope, setScope] = useState("");
  const [scopeDraft, setScopeDraft] = useState("");
  const [selected, setSelected] = useState<string | null>(null);
  const { result, listen } = Picker.useListen<Listing>(
    async (client) => {
      const r = await browseTopics(client, device, toFilter(scope));
      const topics = r.topics
        // Sparkplug B payloads are not JSON. They have a tab of their own.
        .filter((t) => !isSparkplugTopic(t.topic))
        .map((t) => ({ ...t, leaves: leavesOf(t) }));
      return { key: id.create(), topics, truncated: r.truncated };
    },
    [device, scope],
  );
  const listing = result.data;
  const topics = listing?.topics ?? NO_TOPICS;

  const taken = takenBy(entries, topics, single);
  const trimmed = search.trim();
  const q = trimmed.toLowerCase();
  const visible =
    q === "" ? topics : topics.filter((t) => t.topic.toLowerCase().includes(q));
  const folders = buildFolders(visible);
  const paths = byPath(folders);
  const nodes = toNodes(folders);
  const items = toItems(folders, value, taken);
  const initialExpanded = branchPaths(topics);
  const active = selected ?? topics[0]?.topic ?? null;
  const activeTopic = topics.find((t) => t.topic === active) ?? null;

  const handlePick = (path: string) => {
    const folder = paths.get(path);
    if (folder == null) throw new Error(`No topic level at ${path}`);
    const pickable = folder.topics.filter((t) => freeCount(t, taken) > 0);
    const all = pickable.length > 0 && pickable.every((t) => value.has(t.topic));
    const next = new Map(value);
    for (const t of pickable) {
      const pointers = all
        ? new Set<string>()
        : defaultPointers(t.leaves, taken.get(t.topic), single);
      if (pointers.size === 0) next.delete(t.topic);
      else next.set(t.topic, { leaves: t.leaves, pointers });
    }
    onChange(next);
    if (folder.topic != null) setSelected(folder.path);
  };

  let tree: ReactNode;
  if (listing != null)
    tree = (
      <Picker.Tree
        key={listing.key}
        nodes={nodes}
        items={items}
        initialExpanded={initialExpanded}
        active={active}
        onActivate={setSelected}
        onPick={handlePick}
        emptyContent={
          <Empty.Action
            message={q === "" ? NOTHING_PUBLISHED : `No topic matches “${trimmed}”.`}
            className={CSS.BE("mqtt-discover", "notice")}
          />
        }
      />
    );
  else if (result.variant === "loading")
    tree = <Picker.TreeSkeleton items={SKELETON} />;
  else tree = <Status.Summary center status={result.status} />;

  const loading = result.variant === "loading";
  let statusText = pluralize("topic", topics.length, true);
  if (loading) statusText = Picker.LISTENING;
  else if (listing?.truncated === true) statusText += ", list is full";

  let pane: ReactNode;
  if (activeTopic != null)
    pane = (
      <TopicValues
        key={activeTopic.topic}
        topic={activeTopic}
        value={value.get(activeTopic.topic)?.pointers ?? new Set()}
        taken={taken.get(activeTopic.topic) ?? new Set()}
        single={single}
        onChange={(pointers) => {
          const next = new Map(value);
          if (pointers.size === 0) next.delete(activeTopic.topic);
          else next.set(activeTopic.topic, { leaves: activeTopic.leaves, pointers });
          onChange(next);
        }}
      />
    );
  else if (listing == null && loading) pane = <PaneSkeleton noun="Field" />;
  else
    pane = (
      <Empty.Action
        message="Select a topic to see what it sends."
        className={CSS.BE("mqtt-discover", "notice")}
      />
    );

  return (
    <>
      <Picker.Status status={statusText} loading={loading} onListen={listen} />
      <Picker.Side>
        <Input.Text
          value={search}
          onChange={setSearch}
          size="small"
          placeholder="Search topics"
          startContent={<Icon.Search />}
          className={CSS.BE("mqtt-discover", "search")}
        />
        <Flex.Box y empty className={CSS.BE("mqtt-discover", "tree")}>
          {tree}
        </Flex.Box>
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
        <Button.Button
          variant="text"
          size="small"
          className={CSS.BE("mqtt-discover", "add")}
          onClick={() => onAddByName(trimmed)}
        >
          <Icon.Add />
          {trimmed === "" ? "Add a topic by name" : `Add “${trimmed}” as a topic`}
        </Button.Button>
      </Picker.Side>
      <Picker.Pane>{pane}</Picker.Pane>
    </>
  );
};
