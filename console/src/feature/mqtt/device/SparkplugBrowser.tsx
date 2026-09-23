// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, status, type Synnax as Client } from "@synnaxlabs/client";
import {
  Button,
  Component,
  Flex,
  Haul,
  Icon,
  Input,
  List,
  Select,
  Status,
  Synnax,
  Text,
  TimeSpan,
  Tree,
  useCombinedStateAndRef,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import { retrieveScanTask } from "@/feature/mqtt/device/retrieveScanTask";
import { type Device } from "@/feature/mqtt/device/types";
import {
  BROWSE_SPARKPLUG_COMMAND_TYPE,
  type BrowsedSparkplugNode,
  type BrowsedSparkplugTag,
  type SparkplugTagID,
} from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";

export const SPARKPLUG_HAUL_TYPE = "mqtt-sparkplug-tag";

/** A browsed Sparkplug B tag with the edge node it belongs to. */
export interface SparkplugHaulTag
  extends SparkplugTagID, Pick<BrowsedSparkplugTag, "dataType"> {}

export type SparkplugHaulItem = Haul.Item<
  typeof SPARKPLUG_HAUL_TYPE,
  string,
  SparkplugHaulTag
>;

const tagKey = ({ group, edgeNode, device, tag }: SparkplugTagID): string =>
  `${group}/${edgeNode}/${device}/${tag}`;

export const createSparkplugHaulItem = ({
  group,
  edgeNode,
  device,
  tag,
  dataType,
}: SparkplugHaulTag): SparkplugHaulItem => ({
  type: SPARKPLUG_HAUL_TYPE,
  key: tagKey({ group, edgeNode, device, tag }),
  data: { group, edgeNode, device, tag, dataType },
});

export const isSparkplugHaulItem = (item: Haul.Item): item is SparkplugHaulItem =>
  item.type === SPARKPLUG_HAUL_TYPE;

export const filterSparkplugHaulItems = (items: Haul.Item[]): SparkplugHaulItem[] =>
  items.filter(isSparkplugHaulItem);

export const canDropSparkplugHaulItem =
  Haul.canDropOfType<SparkplugHaulItem>(SPARKPLUG_HAUL_TYPE);

interface NodeItem extends BrowsedSparkplugNode {
  key: string;
  kind: "node";
  loading: boolean;
}

interface DeviceItem {
  key: string;
  kind: "device";
  name: string;
}

interface TagItem
  extends SparkplugHaulTag, Pick<BrowsedSparkplugTag, "value" | "supported"> {
  key: string;
  kind: "tag";
}

type Item = NodeItem | DeviceItem | TagItem;

const isDraggable = (item: Item): item is TagItem =>
  item.kind === "tag" && item.supported;

const nodeKey = ({ group, edgeNode }: BrowsedSparkplugNode): string =>
  `${group}/${edgeNode}`;

const deviceKey = (node: BrowsedSparkplugNode, device: string): string =>
  `${nodeKey(node)}/${device}`;

const itemRenderProp = Component.renderProp((props: Tree.ItemRenderProps<string>) => {
  const { itemKey } = props;
  const item = List.useItem<string, Item>(itemKey);
  const { getState } = Select.useContext<string>();
  const { getItem } = List.useUtilContext<string, Item>();
  const { startDrag } = Haul.useDrag({ type: SPARKPLUG_HAUL_TYPE, key: itemKey });
  const handleDragStart = useCallback(() => {
    if (item == null || !isDraggable(item)) return;
    const selected = array.toArray(getState().value);
    const dragged =
      getItem != null && selected.includes(itemKey)
        ? getItem(selected).filter(isDraggable)
        : [item];
    startDrag(dragged.map(createSparkplugHaulItem));
  }, [startDrag, item, getState, getItem, itemKey]);
  if (item == null) return null;
  if (item.kind === "node")
    return (
      <Tree.Item {...props} loading={item.loading}>
        <Text.Text color={10} gap="small" overflow="ellipsis">
          <Icon.Node />
          {item.group}/{item.edgeNode}
        </Text.Text>
      </Tree.Item>
    );
  if (item.kind === "device")
    return (
      <Tree.Item {...props}>
        <Text.Text color={10} gap="small" overflow="ellipsis">
          <Icon.Device />
          {item.name}
        </Text.Text>
      </Tree.Item>
    );
  return (
    <Tree.Item
      {...props}
      justify="between"
      disabled={!item.supported}
      draggable={item.supported}
      onDragStart={handleDragStart}
    >
      <Text.Text color={10} gap="small" overflow="ellipsis">
        <Icon.Variable />
        {item.tag}
      </Text.Text>
      <Text.Text level="small" color={9} overflow="ellipsis">
        {item.value === "" ? item.dataType : `${item.value} · ${item.dataType}`}
      </Text.Text>
    </Tree.Item>
  );
});

const BROWSE_DURATION = TimeSpan.seconds(3);

// The driver listens for up to 20 s before it replies.
const BROWSE_TIMEOUT = TimeSpan.seconds(25);

interface BrowseArgs {
  group: string;
  edgeNode: string;
}

const browse = async (
  client: Client,
  device: Device,
  { group, edgeNode }: BrowseArgs,
) => {
  const scanTask = await retrieveScanTask(client, device.rack);
  const { details, variant, message } = await scanTask.executeCommandSync({
    type: BROWSE_SPARKPLUG_COMMAND_TYPE,
    timeout: BROWSE_TIMEOUT,
    args: {
      device: device.key,
      group,
      edge_node: edgeNode,
      duration: BROWSE_DURATION.milliseconds,
    },
  });
  if (variant !== "success") throw new Error(message);
  const data = details?.data;
  if (data == null || !("nodes" in data)) return { nodes: [], tags: [] };
  return data;
};

const GROUP_INPUT_PROPS = { placeholder: "All groups" } as const;

const EMPTY_CONTENT = (
  <Text.Text center status="disabled" className={CSS.BE("mqtt-browser", "empty")}>
    No edge nodes. Browse the broker to list the edge nodes that publish now.
  </Text.Text>
);

export interface SparkplugBrowserProps {
  device: Device;
}

export const SparkplugBrowser = ({ device }: SparkplugBrowserProps) => {
  const [group, setGroup] = useState("");
  const [nodes, setNodes, nodesRef] = useCombinedStateAndRef<Tree.Node[]>([]);
  const store = List.useMapData<string, Item>();
  const [stat, setStat] = useState<status.Status | null>(null);
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();

  const browseNodes = useCallback(() => {
    setNodes([]);
    setStat(status.create({ variant: "loading", message: "Browsing edge nodes" }));
    handleError(async () => {
      let browsed: BrowsedSparkplugNode[];
      try {
        if (client == null) throw new DisconnectedError();
        ({ nodes: browsed } = await browse(client, device, { group, edgeNode: "" }));
      } catch (e) {
        setStat(status.fromException(e, "Failed to browse edge nodes"));
        return;
      }
      store.setItem(
        browsed.map((n) => ({ ...n, key: nodeKey(n), kind: "node", loading: false })),
      );
      setNodes(browsed.map((n) => ({ key: nodeKey(n), children: [] })));
      setStat(status.create({ variant: "success", message: "Browsed edge nodes" }));
    }, "Failed to browse edge nodes");
  }, [client, device, group, handleError, store, setNodes]);

  const handleExpand = useCallback(
    ({ clicked, action }: Tree.HandleExpandProps) => {
      const node = store.getItem(clicked);
      if (action === "contract" || node?.kind !== "node") return;
      store.setItem({ ...node, loading: true });
      handleError(async () => {
        try {
          if (client == null) throw new DisconnectedError();
          const { tags } = await browse(client, device, node);
          const items: Item[] = [];
          const children: Tree.Node[] = [];
          tags.forEach(({ name: tag, device: owner, ...rest }) => {
            const { group, edgeNode } = node;
            const id = { group, edgeNode, device: owner, tag };
            const item: TagItem = { ...rest, ...id, key: tagKey(id), kind: "tag" };
            items.push(item);
            if (owner === "") {
              children.push({ key: item.key });
              return;
            }
            const key = deviceKey(node, owner);
            let parent = children.find((c) => c.key === key);
            if (parent == null) {
              parent = { key, children: [] };
              children.push(parent);
              items.push({ key, kind: "device", name: owner });
            }
            parent.children?.push({ key: item.key });
          });
          store.setItem(items);
          setNodes([
            ...Tree.updateNodeChildren({
              tree: nodesRef.current,
              parent: clicked,
              updater: () => children,
              throwOnMissing: false,
            }),
          ]);
          setStat(status.create({ variant: "success", message: "Browsed tags" }));
        } catch (e) {
          setStat(status.fromException(e, "Failed to browse tags"));
        } finally {
          store.setItem({ ...node, loading: false });
        }
      }, "Failed to browse tags");
    },
    [client, device, handleError, store, setNodes, nodesRef],
  );

  const treeProps = Tree.use({ nodes, onExpand: handleExpand });
  const loading = stat?.variant === "loading";
  const failed = stat?.variant === "error";
  let content: ReactElement;
  if (failed && nodes.length === 0) content = <Status.Summary center status={stat} />;
  else if (loading)
    content = (
      <Flex.Box center>
        <Icon.Loading className={CSS.BE("mqtt-browser", "loading-icon")} color={9} />
      </Flex.Box>
    );
  else
    content = (
      <Tree.Tree
        {...treeProps}
        getItem={store.getItem}
        subscribe={store.subscribe}
        emptyContent={EMPTY_CONTENT}
      >
        {itemRenderProp}
      </Tree.Tree>
    );
  return (
    <>
      <Flex.Box x className={CSS.BE("mqtt-browser", "controls")}>
        <Input.Text value={group} onChange={setGroup} {...GROUP_INPUT_PROPS} />
        <Button.Button onClick={browseNodes} disabled={loading} variant="filled">
          Browse
        </Button.Button>
      </Flex.Box>
      {failed && nodes.length > 0 && (
        <Status.Summary status={stat} className={CSS.BE("mqtt-browser", "note")} />
      )}
      {content}
    </>
  );
};
