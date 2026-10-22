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
import { type record } from "@synnaxlabs/x";
import {
  type Dispatch,
  type ReactElement,
  type ReactNode,
  type SetStateAction,
  useCallback,
  useMemo,
  useState,
} from "react";

import { browseSparkplug } from "@/feature/mqtt/device/browse";
import { type SparkplugHaulTag, tagKey } from "@/feature/mqtt/device/SparkplugBrowser";
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
import {
  type BrowsedSparkplugNode,
  type BrowsedSparkplugTag,
  type ReadEntry,
} from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";

export type TagPicks = Map<string, SparkplugHaulTag>;

interface Listing {
  nodes: BrowsedSparkplugNode[];
  /** Counts listens, so each one opens a fresh, fully expanded tree. */
  version: number;
}

interface NodeTags {
  tags: BrowsedSparkplugTag[] | null;
  loading: boolean;
}

const nodeKey = ({ group, edgeNode }: BrowsedSparkplugNode) => `${group}/${edgeNode}`;

const toTag = (
  { group, edgeNode }: BrowsedSparkplugNode,
  t: BrowsedSparkplugTag,
): SparkplugHaulTag => ({
  group,
  edgeNode,
  device: t.device,
  tag: t.name,
  dataType: t.dataType,
});

const inNode = (tag: SparkplugHaulTag, node: BrowsedSparkplugNode) =>
  tag.group === node.group && tag.edgeNode === node.edgeNode;

interface GroupItemProps extends Tree.ItemRenderProps<string> {
  count: number;
}

const GroupItem = ({ count, ...props }: GroupItemProps) => (
  <Tree.Item {...props} className={CSS.BE("mqtt-discover", "item")}>
    <Text.Text
      level="small"
      weight={500}
      color={10}
      overflow="ellipsis"
      className={CSS.BE("mqtt-discover", "code")}
    >
      {props.itemKey}
    </Text.Text>
    <Text.Text level="small" className={CSS.BE("mqtt-discover", "meta")}>
      {plural(count, "edge node")}
    </Text.Text>
  </Tree.Item>
);

interface NodeItemProps extends Tree.ItemRenderProps<string> {
  node: BrowsedSparkplugNode;
  state: NodeTags | undefined;
  picked: number;
  free: number | null;
  onPick: (node: BrowsedSparkplugNode) => void;
}

const NodeItem = ({ node, state, picked, free, onPick, ...props }: NodeItemProps) => {
  const all = picked > 0 && free != null && picked >= free;
  let meta = "";
  if (picked > 0) meta = `${picked} selected`;
  else if (state?.tags != null) meta = plural(state.tags.length, "tag");
  return (
    <Tree.Item
      {...props}
      loading={state?.loading}
      className={CSS.BE("mqtt-discover", "item")}
    >
      <Input.Checkbox
        value={all}
        indeterminate={picked > 0 && !all}
        onChange={() => onPick(node)}
        disabled={state?.loading === true}
        size="small"
        aria-label={`Pick ${node.edgeNode}`}
      />
      <Text.Text
        level="small"
        weight={500}
        color={10}
        overflow="ellipsis"
        className={CSS.BE("mqtt-discover", "code")}
      >
        {node.edgeNode}
      </Text.Text>
      <Text.Text
        level="small"
        className={CSS.cls(
          CSS.BE("mqtt-discover", "meta"),
          picked > 0 && CSS.M("picked"),
        )}
      >
        {meta}
      </Text.Text>
    </Tree.Item>
  );
};

interface NodeTreeProps {
  nodes: BrowsedSparkplugNode[];
  tags: Map<string, NodeTags>;
  active: string | null;
  onActivate: (node: BrowsedSparkplugNode) => void;
  pickedIn: (node: BrowsedSparkplugNode) => number;
  freeIn: (node: BrowsedSparkplugNode) => number | null;
  onPick: (node: BrowsedSparkplugNode) => void;
}

const NodeTree = ({
  nodes,
  tags,
  active,
  onActivate,
  pickedIn,
  freeIn,
  onPick,
}: NodeTreeProps) => {
  const byKey = useMemo(() => new Map(nodes.map((n) => [nodeKey(n), n])), [nodes]);
  const groups = useMemo(() => {
    const m = new Map<string, BrowsedSparkplugNode[]>();
    for (const n of nodes) m.set(n.group, [...(m.get(n.group) ?? []), n]);
    return m;
  }, [nodes]);
  const treeNodes = useMemo<Tree.Node[]>(
    () =>
      [...groups].map(([group, members]) => ({
        key: group,
        children: members.map((n) => ({ key: nodeKey(n) })),
      })),
    [groups],
  );
  const tree = Tree.use({
    nodes: treeNodes,
    initialExpanded: [...groups.keys()],
    toggleOn: "caret",
  });
  const { toggle } = tree;
  const handleSelect = useCallback<
    Tree.TreeProps<string, record.Keyed<string>>["onSelect"]
  >(
    (_, { clicked }) => {
      if (clicked == null) return;
      const node = byKey.get(clicked);
      if (node == null) toggle(clicked);
      else onActivate(node);
    },
    [byKey, onActivate, toggle],
  );
  const renderItem = useCallback<Component.RenderProp<Tree.ItemRenderProps<string>>>(
    ({ key, ...props }) => {
      const node = byKey.get(props.itemKey);
      if (node == null)
        return (
          <GroupItem
            key={key}
            {...props}
            count={groups.get(props.itemKey)?.length ?? 0}
          />
        );
      return (
        <NodeItem
          key={key}
          {...props}
          node={node}
          state={tags.get(props.itemKey)}
          picked={pickedIn(node)}
          free={freeIn(node)}
          onPick={onPick}
        />
      );
    },
    [byKey, groups, tags, pickedIn, freeIn, onPick],
  );
  return (
    <Tree.Tree<string, record.Keyed<string>>
      {...tree}
      selected={active == null ? [] : [active]}
      onSelect={handleSelect}
      itemHeight={ITEM_HEIGHT}
    >
      {renderItem}
    </Tree.Tree>
  );
};

const SKELETON: SkeletonItem[] = [
  [0, "12ch"],
  [1, "17ch"],
  [1, "14ch"],
];

export interface TagPickerProps {
  device: Device;
  entries: ReadEntry[];
  hidden: boolean;
  tabs: ReactElement;
  picks: TagPicks;
  onPicksChange: Dispatch<SetStateAction<TagPicks>>;
  onAddByName: () => void;
}

/** Lists the Sparkplug B edge nodes on the broker and the tags of the one selected. */
export const TagPicker = ({
  device,
  entries,
  hidden,
  tabs,
  picks,
  onPicksChange,
  onAddByName,
}: TagPickerProps) => {
  const client = Synnax.use();
  const [listing, setListing] = useState<Listing | null>(null);
  const [tags, setTags] = useState(new Map<string, NodeTags>());
  const [error, setError] = useState<status.Status | null>(null);
  const [loading, setLoading] = useState(false);
  const [round, setRound] = useState(0);
  const [active, setActive] = useState<string | null>(null);

  const taken = useMemo(
    () => new Set(entries.flatMap((e) => (e.type === "sparkplug" ? [tagKey(e)] : []))),
    [entries],
  );

  const loadTags = useCallback(
    async (node: BrowsedSparkplugNode): Promise<BrowsedSparkplugTag[]> => {
      const key = nodeKey(node);
      setTags((prev) => new Map(prev).set(key, { tags: null, loading: true }));
      try {
        if (client == null) throw new DisconnectedError();
        const { tags } = await browseSparkplug(client, device, node);
        setTags((prev) => new Map(prev).set(key, { tags, loading: false }));
        return tags;
      } catch (e) {
        setError(
          status.fromException(e, `Could not ask ${node.edgeNode} for its tags`),
        );
        setTags((prev) => new Map(prev).set(key, { tags: null, loading: false }));
        return [];
      }
    },
    [client, device],
  );

  useAsyncEffect(
    async (signal) => {
      setLoading(true);
      setError(null);
      try {
        if (client == null) throw new DisconnectedError();
        const { nodes } = await browseSparkplug(client, device, {
          group: "",
          edgeNode: "",
        });
        if (signal.aborted) return;
        setListing((prev) => ({ nodes, version: (prev?.version ?? 0) + 1 }));
        setTags(new Map());
        const first = nodes[0];
        if (first != null) {
          setActive(nodeKey(first));
          void loadTags(first);
        }
      } catch (e) {
        if (!signal.aborted) setError(status.fromException(e, "Could not listen"));
      } finally {
        if (!signal.aborted) setLoading(false);
      }
    },
    [client, device, loadTags, round],
  );

  const freeTags = useCallback(
    (node: BrowsedSparkplugNode, list: BrowsedSparkplugTag[]) =>
      list
        .filter((t) => t.supported)
        .map((t) => toTag(node, t))
        .filter((t) => !taken.has(tagKey(t))),
    [taken],
  );
  const pickedIn = useCallback(
    (node: BrowsedSparkplugNode) => [...picks.values()].filter((t) => inNode(t, node)),
    [picks],
  );
  const setNodePicks = useCallback(
    (node: BrowsedSparkplugNode, picked: SparkplugHaulTag[]) =>
      onPicksChange((prev) => {
        const next = new Map(prev);
        for (const [key, t] of prev) if (inNode(t, node)) next.delete(key);
        for (const t of picked) next.set(tagKey(t), t);
        return next;
      }),
    [onPicksChange],
  );
  const handlePick = useCallback(
    async (node: BrowsedSparkplugNode) => {
      setActive(nodeKey(node));
      const list = tags.get(nodeKey(node))?.tags ?? (await loadTags(node));
      const free = freeTags(node, list);
      const all = free.length > 0 && pickedIn(node).length >= free.length;
      setNodePicks(node, all ? [] : free);
    },
    [tags, loadTags, freeTags, pickedIn, setNodePicks],
  );
  const handleActivate = useCallback(
    (node: BrowsedSparkplugNode) => {
      const key = nodeKey(node);
      setActive(key);
      const state = tags.get(key);
      if (state?.tags == null && state?.loading !== true) void loadTags(node);
    },
    [tags, loadTags],
  );

  const nodes = listing?.nodes ?? [];
  const activeNode = nodes.find((n) => nodeKey(n) === active) ?? null;
  const activeTags = active == null ? undefined : tags.get(active);
  const asking = nodes.find((n) => tags.get(nodeKey(n))?.loading === true);
  let statusText = plural(nodes.length, "edge node");
  if (loading) statusText = LISTENING;
  else if (asking != null) statusText = `Asking ${asking.edgeNode} for its tags`;

  let tree: ReactNode;
  if (error != null && nodes.length === 0)
    tree = <Status.Summary center status={error} />;
  else if (listing == null) tree = <TreeSkeleton items={SKELETON} />;
  else if (nodes.length === 0)
    tree = (
      <Empty.Action
        message="No Sparkplug B edge node published while Synnax listened."
        className={CSS.BE("mqtt-discover", "notice")}
      />
    );
  else
    tree = (
      <NodeTree
        key={listing.version}
        nodes={nodes}
        tags={tags}
        active={active}
        onActivate={handleActivate}
        pickedIn={(node) => pickedIn(node).length}
        freeIn={(node) => {
          const list = tags.get(nodeKey(node))?.tags;
          return list == null ? null : freeTags(node, list).length;
        }}
        onPick={(node) => void handlePick(node)}
      />
    );

  let pane: ReactNode;
  if (activeNode == null)
    pane =
      listing == null ? (
        <PaneSkeleton noun="Tag" />
      ) : (
        <Empty.Action
          message="Select an edge node to see its tags."
          className={CSS.BE("mqtt-discover", "notice")}
        />
      );
  else if (activeTags == null || activeTags.loading)
    pane = (
      <PaneSkeleton noun="Tag" title={`${activeNode.group} / ${activeNode.edgeNode}`} />
    );
  else if (activeTags.tags == null)
    pane = (
      <Status.Summary
        center
        status={
          error ??
          status.create({
            variant: "error",
            message: `Could not ask ${activeNode.edgeNode} for its tags`,
          })
        }
      />
    );
  else {
    const node = activeNode;
    const list = activeTags.tags;
    const byKey = new Map(list.map((t) => [tagKey(toTag(node, t)), toTag(node, t)]));
    const byDevice = new Map<string, Value[]>();
    for (const t of list) {
      const key = tagKey(toTag(node, t));
      byDevice.set(t.device, [
        ...(byDevice.get(t.device) ?? []),
        {
          key,
          name: t.name,
          sample: t.value,
          dataType: t.dataType,
          taken: taken.has(key),
          unsupported: !t.supported,
        },
      ]);
    }
    const free = freeTags(node, list);
    const picked = pickedIn(node).map(tagKey);
    pane = (
      <>
        <PaneHeader
          title={`${node.group} / ${node.edgeNode}`}
          free={free.length}
          picked={picked.length}
          onPickAll={() => setNodePicks(node, free)}
          onClear={() => setNodePicks(node, [])}
        />
        <Values
          noun="Tag"
          groups={[...byDevice].map(([owner, values]) => ({
            title: owner === "" ? "Edge node" : owner,
            values,
          }))}
          picked={picked}
          onPickedChange={(keys) =>
            setNodePicks(
              node,
              keys.flatMap((k) => {
                const t = byKey.get(k);
                return t == null ? [] : [t];
              }),
            )
          }
        />
      </>
    );
  }

  const side = (
    <>
      <Flex.Box y empty className={CSS.BE("mqtt-discover", "tree")}>
        {tree}
      </Flex.Box>
      <Button.Button
        variant="text"
        size="small"
        className={CSS.BE("mqtt-discover", "add")}
        onClick={onAddByName}
      >
        <Icon.Add />
        Add a tag by name
      </Button.Button>
    </>
  );

  return (
    <Picker
      tabs={tabs}
      status={statusText}
      loading={loading || asking != null}
      hidden={hidden}
      onRefresh={() => setRound((r) => r + 1)}
      side={side}
      pane={pane}
    />
  );
};
