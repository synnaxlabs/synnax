// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { type Tree } from "@synnaxlabs/lyra/tree";
import { id, map } from "@synnaxlabs/x";
import pluralize from "pluralize";
import { type ReactNode, useEffect, useState } from "react";

import {
  BROWSE_DURATION,
  browseSparkplug,
  LISTENING,
} from "@/feature/mqtt/device/browse";
import { type Device } from "@/feature/mqtt/device/types";
import {
  type BrowsedSparkplugNode,
  type BrowsedSparkplugTag,
  type HeldEntry,
  nodeKey,
  type SparkplugTag,
  tagKey,
} from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { ICD } from "@/platform/icd";

export type TagPicks = Map<string, SparkplugTag>;

interface Listing {
  /** Names the listen, so each one opens a fresh, fully expanded tree. */
  key: string;
  nodes: BrowsedSparkplugNode[];
}

interface NodeTags {
  tags: BrowsedSparkplugTag[] | null;
  loading: boolean;
  error?: status.Status;
}

/** The tags of each edge node, asked for after the listen that found the nodes. */
interface ListingTags {
  listing: string | null;
  nodes: Map<string, NodeTags>;
}

const ALL_NODES = { group: "", edgeNode: "" };

const NO_NODES: BrowsedSparkplugNode[] = [];

const toTag = (
  { group, edgeNode }: BrowsedSparkplugNode,
  t: BrowsedSparkplugTag,
): SparkplugTag => ({
  group,
  edgeNode,
  device: t.device,
  tag: t.name,
  dataType: t.dataType,
});

const inNode = (tag: SparkplugTag, node: BrowsedSparkplugNode) =>
  tag.group === node.group && tag.edgeNode === node.edgeNode;

const pickedIn = (picks: TagPicks, node: BrowsedSparkplugNode): SparkplugTag[] =>
  [...picks.values()].filter((t) => inNode(t, node));

const replaceNodePicks = (
  picks: TagPicks,
  node: BrowsedSparkplugNode,
  picked: SparkplugTag[],
): TagPicks => {
  const next = new Map(picks);
  for (const [key, t] of picks) if (inNode(t, node)) next.delete(key);
  for (const t of picked) next.set(tagKey(t), t);
  return next;
};

const pickedOf = (
  picked: number,
  free: number | null,
): ICD.Picker.TreeItem["picked"] => {
  if (picked === 0) return "none";
  return free != null && picked >= free ? "all" : "some";
};

const SKELETON: ICD.Picker.SkeletonItem[] = [
  [0, "12ch"],
  [1, "17ch"],
  [1, "14ch"],
];

export interface TagPickerProps {
  device: Device;
  /** What the task holds now. */
  entries: HeldEntry[];
  value: TagPicks;
  onChange: (value: TagPicks) => void;
  onAddByName: () => void;
}

/** Lists the Sparkplug B edge nodes on the broker and the tags of the one selected. */
export const TagPicker = ({
  device,
  entries,
  value,
  onChange,
  onAddByName,
}: TagPickerProps) => {
  const [selected, setSelected] = useState<string | null>(null);
  const [tags, setTags] = useState<ListingTags>(() => ({
    listing: null,
    nodes: new Map(),
  }));
  // handlePick reads the picks made while it waits for tags.
  const valueRef = useSyncedRef(value);
  const { result, listen, follow } = ICD.Picker.useListen<Listing>(
    async (client) => {
      const { nodes } = await browseSparkplug(client, device, ALL_NODES);
      return { key: id.create(), nodes };
    },
    [device],
  );
  const listing = result.data;
  if (listing != null && tags.listing !== listing.key)
    setTags({ listing: listing.key, nodes: new Map() });

  const loadTags = async (
    node: BrowsedSparkplugNode,
  ): Promise<BrowsedSparkplugTag[] | null> => {
    const key = nodeKey(node);
    const set = (state: NodeTags) =>
      setTags((prev) => ({ ...prev, nodes: new Map(prev.nodes).set(key, state) }));
    set({ tags: null, loading: true });
    try {
      const answer = await follow((client) => browseSparkplug(client, device, node));
      if (answer == null) return null;
      set({ tags: answer.tags, loading: false });
      return answer.tags;
    } catch (e) {
      set({
        tags: null,
        loading: false,
        error: status.fromException(e, {
          message: "Failed to get tags",
          name: node.edgeNode,
        }),
      });
      return null;
    }
  };

  const nodes = listing?.nodes ?? NO_NODES;
  const first = nodes.at(0);
  const active = selected ?? (first == null ? null : nodeKey(first));
  const activeNode = nodes.find((n) => nodeKey(n) === active) ?? null;
  const activeTags = active == null ? undefined : tags.nodes.get(active);
  useEffect(() => {
    if (activeNode != null && activeTags == null) void loadTags(activeNode);
  }, [activeNode, activeTags, loadTags]);

  const taken = new Set(
    entries.flatMap((e) => (e.type === "sparkplug" ? [tagKey(e)] : [])),
  );
  const freeTags = (node: BrowsedSparkplugNode, list: BrowsedSparkplugTag[]) =>
    list
      .filter((t) => t.supported)
      .map((t) => toTag(node, t))
      .filter((t) => !taken.has(tagKey(t)));
  const byKey = new Map(nodes.map((n) => [nodeKey(n), n]));
  const groups = Map.groupBy(nodes, (n) => n.group);
  const treeNodes: Tree.Node[] = [...groups].map(([group, members]) => ({
    key: group,
    children: members.map((n) => ({ key: nodeKey(n) })),
  }));
  const initialExpanded = [...groups.keys()];
  const items: ICD.Picker.TreeItem[] = [...groups].flatMap(([group, members]) => [
    {
      key: group,
      name: group,
      meta: pluralize("edge node", members.length, true),
      folder: true,
    },
    ...members.map((node): ICD.Picker.TreeItem => {
      const key = nodeKey(node);
      const state = tags.nodes.get(key);
      const picked = pickedIn(value, node).length;
      const free = state?.tags == null ? null : freeTags(node, state.tags).length;
      let meta = "";
      if (picked > 0) meta = `${picked} selected`;
      else if (state?.tags != null) meta = pluralize("tag", state.tags.length, true);
      return {
        key,
        name: node.edgeNode,
        meta,
        picked: pickedOf(picked, free),
        disabled: state?.loading === true,
        loading: state?.loading === true,
      };
    }),
  ]);

  const handlePick = async (key: string) => {
    const node = byKey.get(key);
    if (node == null) throw new Error(`No edge node at ${key}`);
    setSelected(key);
    const list = tags.nodes.get(key)?.tags ?? (await loadTags(node));
    if (list == null) return;
    const free = freeTags(node, list);
    const picks = valueRef.current;
    const all = free.length > 0 && pickedIn(picks, node).length >= free.length;
    onChange(replaceNodePicks(picks, node, all ? [] : free));
  };
  const handleActivate = (key: string) => {
    if (byKey.has(key)) setSelected(key);
  };

  const loading = result.variant === "loading";
  const asking = nodes.find((n) => tags.nodes.get(nodeKey(n))?.loading === true);
  let statusText = pluralize("edge node", nodes.length, true);
  if (loading) statusText = LISTENING;
  else if (asking != null) statusText = `Asking ${asking.edgeNode} for its tags`;

  let tree: ReactNode;
  if (listing != null)
    tree = (
      <ICD.Picker.Tree
        key={listing.key}
        nodes={treeNodes}
        items={items}
        initialExpanded={initialExpanded}
        active={active}
        onActivate={handleActivate}
        onPick={(key) => void handlePick(key)}
        emptyContent={
          <Empty.Action
            message="No Sparkplug B edge node published while Synnax listened."
            className={CSS.BE("mqtt-discover", "notice")}
          />
        }
      />
    );
  else if (loading) tree = <ICD.Picker.TreeSkeleton items={SKELETON} />;
  else tree = <Status.Summary center status={result.status} />;

  let pane: ReactNode;
  if (activeNode == null)
    pane =
      listing == null && loading ? (
        <ICD.Picker.PaneSkeleton noun="Tag" />
      ) : (
        <Empty.Action
          message="Select an edge node to see its tags."
          className={CSS.BE("mqtt-discover", "notice")}
        />
      );
  else if (activeTags == null || activeTags.loading)
    pane = (
      <ICD.Picker.PaneSkeleton
        noun="Tag"
        title={`${activeNode.group} / ${activeNode.edgeNode}`}
      />
    );
  else if (activeTags.tags == null)
    pane = <Status.Summary center status={activeTags.error} />;
  else {
    const node = activeNode;
    const list = activeTags.tags;
    const tagsByKey = new Map(
      list.map((t) => [tagKey(toTag(node, t)), toTag(node, t)]),
    );
    const byDevice = new Map<string, ICD.Picker.Value[]>();
    for (const t of list) {
      const key = tagKey(toTag(node, t));
      map
        .getOrInsert(byDevice, t.device, () => [])
        .push({
          key,
          name: t.name,
          sample: t.value,
          dataType: t.dataType,
          taken: taken.has(key),
          unsupported: !t.supported,
        });
    }
    const free = freeTags(node, list);
    const picked = pickedIn(value, node).map(tagKey);
    pane = (
      <>
        <ICD.Picker.PaneHeader
          title={`${node.group} / ${node.edgeNode}`}
          free={free.length}
          picked={picked.length}
          onPickAll={() => onChange(replaceNodePicks(value, node, free))}
          onClear={() => onChange(replaceNodePicks(value, node, []))}
        />
        <ICD.Picker.Values
          noun="Tag"
          groups={[...byDevice].map(([owner, values]) => ({
            title: owner === "" ? "Edge node" : owner,
            values,
          }))}
          value={picked}
          onChange={(keys) =>
            onChange(
              replaceNodePicks(
                value,
                node,
                keys.flatMap((k) => {
                  const t = tagsByKey.get(k);
                  return t == null ? [] : [t];
                }),
              ),
            )
          }
        />
      </>
    );
  }

  return (
    <>
      <ICD.Picker.Status
        duration={BROWSE_DURATION}
        status={statusText}
        loading={loading || asking != null}
        onListen={listen}
      />
      <ICD.Picker.Side>
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
      </ICD.Picker.Side>
      <ICD.Picker.Pane>{pane}</ICD.Picker.Pane>
    </>
  );
};
