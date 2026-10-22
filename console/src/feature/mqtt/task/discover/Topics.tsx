// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { type Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree } from "@synnaxlabs/lyra/tree";
import { Synnax } from "@synnaxlabs/pluto";
import { type record, unique } from "@synnaxlabs/x";
import {
  type Dispatch,
  type ReactElement,
  type ReactNode,
  type SetStateAction,
  useCallback,
  useMemo,
  useState,
} from "react";

import { browseTopics, type KeyedTopic } from "@/feature/mqtt/device/browse";
import { type Device } from "@/feature/mqtt/device/types";
import {
  ITEM_HEIGHT,
  LISTENING,
  Picker,
  plural,
  type SkeletonItem,
  TreeSkeleton,
} from "@/feature/mqtt/task/discover/Picker";
import {
  PaneHeader,
  PaneSkeleton,
  type Value,
  Values,
} from "@/feature/mqtt/task/discover/Values";
import { leavesOf, pointerLabel, sampleAt } from "@/feature/mqtt/task/entries";
import { isSparkplugTopic } from "@/feature/mqtt/task/sparkplug";
import { type ReadEntry, type ReadField } from "@/feature/mqtt/task/types";
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

interface Listing {
  topics: ListedTopic[];
  truncated: boolean;
  /** Counts listens, so each one opens a fresh, fully expanded tree. */
  version: number;
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

/** The values a click on a topic picks: its numbers and booleans, or all of them. */
const defaultPointers = (leaves: ReadField[], taken?: Set<string>): Set<string> => {
  const free = leaves.filter((l) => taken?.has(l.pointer) !== true);
  const scalars = free.filter((l) => !l.disabled);
  return new Set((scalars.length > 0 ? scalars : free).map((l) => l.pointer));
};

/** The values picked from one topic, with the fields they came from. */
export interface TopicPick {
  leaves: ReadField[];
  pointers: Set<string>;
}

export type TopicPicks = Map<string, TopicPick>;

/** The fields of each topic the task reads already. */
type Taken = Map<string, Set<string>>;

const freeCount = (t: ListedTopic, taken: Taken) =>
  t.leaves.filter((l) => taken.get(t.topic)?.has(l.pointer) !== true).length;

interface ItemProps extends Tree.ItemRenderProps<string> {
  folder: Folder;
  picks: TopicPicks;
  taken: Taken;
  onPick: (folder: Folder) => void;
}

const Item = ({ folder, picks, taken, onPick, ...props }: ItemProps) => {
  const { topic, topics } = folder;
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
  return (
    <Tree.Item {...props} className={CSS.BE("mqtt-discover", "item")}>
      <Input.Checkbox
        value={all}
        indeterminate={!all && values > 0}
        onChange={() => onPick(folder)}
        disabled={pickable.length === 0}
        size="small"
        aria-label={`Pick ${folder.name}`}
      />
      <Text.Text
        level="small"
        weight={500}
        color={10}
        overflow="ellipsis"
        className={CSS.BE("mqtt-discover", "code")}
      >
        {folder.name === "" ? "(empty level)" : folder.name}
      </Text.Text>
      <Text.Text
        level="small"
        className={CSS.cls(
          CSS.BE("mqtt-discover", "meta"),
          values > 0 && CSS.M("picked"),
        )}
      >
        {meta}
      </Text.Text>
    </Tree.Item>
  );
};

interface TopicTreeProps {
  folders: Folder[];
  initialExpanded: string[];
  active: string | null;
  onActiveChange: (path: string) => void;
  picks: TopicPicks;
  taken: Taken;
  onPick: (folder: Folder) => void;
  emptyContent: ReactElement;
}

const TopicTree = ({
  folders,
  initialExpanded,
  active,
  onActiveChange,
  picks,
  taken,
  onPick,
  emptyContent,
}: TopicTreeProps) => {
  const paths = useMemo(() => byPath(folders), [folders]);
  const nodes = useMemo(() => toNodes(folders), [folders]);
  const tree = Tree.use({ nodes, initialExpanded, toggleOn: "caret" });
  const { toggle } = tree;
  const handleSelect = useCallback<
    Tree.TreeProps<string, record.Keyed<string>>["onSelect"]
  >(
    (_, { clicked }) => {
      if (clicked == null) return;
      onActiveChange(clicked);
      if (paths.get(clicked)?.topic == null) toggle(clicked);
    },
    [paths, onActiveChange, toggle],
  );
  const renderItem = useCallback<Component.RenderProp<Tree.ItemRenderProps<string>>>(
    ({ key, ...props }) => {
      const folder = paths.get(props.itemKey);
      if (folder == null) return null;
      return (
        <Item
          key={key}
          {...props}
          folder={folder}
          picks={picks}
          taken={taken}
          onPick={onPick}
        />
      );
    },
    [paths, picks, taken, onPick],
  );
  return (
    <Tree.Tree<string, record.Keyed<string>>
      {...tree}
      selected={active == null ? [] : [active]}
      onSelect={handleSelect}
      itemHeight={ITEM_HEIGHT}
      emptyContent={emptyContent}
    >
      {renderItem}
    </Tree.Tree>
  );
};

interface TopicValuesProps {
  topic: ListedTopic;
  picked: Set<string>;
  taken: Set<string>;
  onChange: (pointers: Set<string>) => void;
}

const TopicValues = ({ topic, picked, taken, onChange }: TopicValuesProps) => {
  const [raw, setRaw] = useState(false);
  const { leaves } = topic;
  const free = leaves.filter((l) => !taken.has(l.pointer)).map((l) => l.pointer);
  const values: Value[] = leaves.map((l) => ({
    key: l.pointer,
    name: pointerLabel(l.pointer),
    sample: sampleAt(topic.payload, l.pointer),
    dataType: l.dataType,
    taken: taken.has(l.pointer),
  }));
  return (
    <>
      <PaneHeader
        title={topic.topic}
        note={topic.retained ? "Retained" : undefined}
        free={free.length}
        picked={picked.size}
        onPickAll={() => onChange(new Set(free))}
        onClear={() => onChange(new Set())}
      />
      <Values
        noun="Field"
        groups={[{ values }]}
        picked={[...picked]}
        onPickedChange={(keys) => onChange(new Set(keys))}
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
const SKELETON: SkeletonItem[] = [
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
  entries: ReadEntry[];
  hidden: boolean;
  tabs: ReactElement;
  picks: TopicPicks;
  onPicksChange: Dispatch<SetStateAction<TopicPicks>>;
  onAddByName: (topic: string) => void;
}

/** Lists the JSON topics the broker publishes, and the fields of the one selected. */
export const TopicPicker = ({
  device,
  entries,
  hidden,
  tabs,
  picks,
  onPicksChange,
  onAddByName,
}: TopicPickerProps) => {
  const client = Synnax.use();
  const [search, setSearch] = useState("");
  const [scope, setScope] = useState("");
  const [scopeDraft, setScopeDraft] = useState("");
  const [listing, setListing] = useState<Listing | null>(null);
  const [error, setError] = useState<status.Status | null>(null);
  const [loading, setLoading] = useState(false);
  const [round, setRound] = useState(0);
  const [active, setActive] = useState<string | null>(null);

  useAsyncEffect(
    async (signal) => {
      setLoading(true);
      setError(null);
      try {
        if (client == null) throw new DisconnectedError();
        const r = await browseTopics(client, device, toFilter(scope));
        if (signal.aborted) return;
        const topics = r.topics
          // Sparkplug B payloads are not JSON. They have a tab of their own.
          .filter((t) => !isSparkplugTopic(t.topic))
          .map((t) => ({ ...t, leaves: leavesOf(t.payload) }));
        setListing((prev) => ({
          topics,
          truncated: r.truncated,
          version: (prev?.version ?? 0) + 1,
        }));
        setActive((prev) => prev ?? topics[0]?.topic ?? null);
      } catch (e) {
        if (!signal.aborted) setError(status.fromException(e, "Could not listen"));
      } finally {
        if (!signal.aborted) setLoading(false);
      }
    },
    [client, device, scope, round],
  );

  const taken = useMemo<Taken>(() => {
    const m: Taken = new Map();
    for (const e of entries)
      if (e.type === "plain") m.set(e.topic, new Set(e.fields.map((f) => f.pointer)));
    return m;
  }, [entries]);

  const topics = useMemo(() => listing?.topics ?? [], [listing]);
  const trimmed = search.trim();
  const q = trimmed.toLowerCase();
  const visible = useMemo(
    () => (q === "" ? topics : topics.filter((t) => t.topic.toLowerCase().includes(q))),
    [topics, q],
  );
  const folders = useMemo(() => buildFolders(visible), [visible]);
  const initialExpanded = useMemo(() => branchPaths(topics), [topics]);
  const activeTopic = topics.find((t) => t.topic === active) ?? null;

  const handlePick = useCallback(
    (folder: Folder) => {
      const pickable = folder.topics.filter((t) => freeCount(t, taken) > 0);
      const all = pickable.length > 0 && pickable.every((t) => picks.has(t.topic));
      onPicksChange((prev) => {
        const next = new Map(prev);
        for (const t of pickable) {
          const pointers = all
            ? new Set<string>()
            : defaultPointers(t.leaves, taken.get(t.topic));
          if (pointers.size === 0) next.delete(t.topic);
          else next.set(t.topic, { leaves: t.leaves, pointers });
        }
        return next;
      });
      if (folder.topic != null) setActive(folder.path);
    },
    [taken, picks, onPicksChange],
  );

  let tree: ReactNode;
  if (error != null) tree = <Status.Summary center status={error} />;
  else if (listing == null) tree = <TreeSkeleton items={SKELETON} />;
  else
    tree = (
      <TopicTree
        key={listing.version}
        folders={folders}
        initialExpanded={initialExpanded}
        active={active}
        onActiveChange={setActive}
        picks={picks}
        taken={taken}
        onPick={handlePick}
        emptyContent={
          <Empty.Action
            message={q === "" ? NOTHING_PUBLISHED : `No topic matches “${trimmed}”.`}
            className={CSS.BE("mqtt-discover", "notice")}
          />
        }
      />
    );

  let statusText = plural(topics.length, "topic");
  if (loading) statusText = LISTENING;
  else if (listing?.truncated === true) statusText += ", list is full";

  let pane: ReactNode;
  if (activeTopic != null)
    pane = (
      <TopicValues
        key={activeTopic.topic}
        topic={activeTopic}
        picked={picks.get(activeTopic.topic)?.pointers ?? new Set()}
        taken={taken.get(activeTopic.topic) ?? new Set()}
        onChange={(pointers) =>
          onPicksChange((prev) => {
            const next = new Map(prev);
            if (pointers.size === 0) next.delete(activeTopic.topic);
            else next.set(activeTopic.topic, { leaves: activeTopic.leaves, pointers });
            return next;
          })
        }
      />
    );
  else if (listing == null && error == null) pane = <PaneSkeleton noun="Field" />;
  else
    pane = (
      <Empty.Action
        message="Select a topic to see what it sends."
        className={CSS.BE("mqtt-discover", "notice")}
      />
    );

  const side = (
    <>
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
    </>
  );

  return (
    <Picker
      tabs={tabs}
      status={statusText}
      loading={loading}
      hidden={hidden}
      onRefresh={() => setRound((r) => r + 1)}
      side={side}
      pane={pane}
    />
  );
};
