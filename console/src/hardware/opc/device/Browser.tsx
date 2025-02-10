// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Header,
  Icon as PIcon,
  Status,
  Synnax,
  TimeSpan,
  Tree,
} from "@synnaxlabs/pluto";
import { type Optional } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useCallback, useEffect, useState } from "react";

import { type Device } from "@/hardware/opc/device/types";
import {
  SCAN_COMMAND_NAME,
  SCAN_NAME,
  type ScanCommandResult,
} from "@/hardware/opc/task/types";

const ICONS: Record<string, ReactElement> = {
  VariableType: <Icon.Type />,
  Variable: <Icon.Variable />,
  ObjectType: <Icon.Type />,
  Object: <Icon.Group />,
};

const nodeKey = (nodeId: string, parentId: string): string => `${nodeId}---${parentId}`;
const parseNodeID = (key: string): string => key.split("---")[0];

export const HAUL_TYPE = "opc";

export interface BrowserProps {
  device: Device;
}

export const Browser = ({ device }: BrowserProps) => {
  const client = Synnax.use();
  const [nodes, setNodes] = useState<Tree.Node[]>([]);
  const { data: scanTask } = useQuery({
    queryKey: [client?.key],
    queryFn: async () => {
      if (client == null) return null;
      const rck = await client.hardware.racks.retrieve(rack.DEFAULT_CHANNEL_NAME);
      return await rck.retrieveTaskByName(SCAN_NAME);
    },
  });
  const [loading, setLoading] = useState<string>();
  const expand = useMutation({
    mutationFn: async ({
      action,
      delay,
      clicked,
    }: Optional<Tree.HandleExpandProps, "clicked"> & { delay?: number }) => {
      if (scanTask?.key == null || action === "contract") return;
      if (delay != null) await new Promise((resolve) => setTimeout(resolve, delay));
      const isRoot = clicked == null;
      const nodeID = isRoot ? "" : parseNodeID(clicked);
      const { connection } = device.properties;
      setLoading(clicked);
      const { details } = await scanTask.executeCommandSync<ScanCommandResult>(
        SCAN_COMMAND_NAME,
        { connection, node_id: nodeID },
        TimeSpan.seconds(10),
      );
      if (details == null) return;
      const { channels } = details;
      const newNodes = channels.map((node) => ({
        key: nodeKey(node.nodeId, nodeID),
        name: node.name,
        icon: node.isArray ? (
          <PIcon.Icon bottomRight={<Icon.Array />}>
            <Icon.Variable />
          </PIcon.Icon>
        ) : (
          ICONS[node.nodeClass]
        ),
        hasChildren: true,
        haulItems: [{ key: node.nodeId, type: HAUL_TYPE, data: node }],
      }));
      setLoading(undefined);
      setInitialLoading(false);
      if (isRoot) setNodes(newNodes);
      else
        setNodes([
          ...Tree.setNode({
            tree: [...nodes],
            destination: clicked,
            additions: newNodes,
          }),
        ]);
    },
  });
  const treeProps = Tree.use({ nodes, onExpand: expand.mutate });
  const [initialLoading, setInitialLoading] = useState(false);
  const refresh = useCallback(() => {
    if (scanTask == null) return;
    setInitialLoading(true);
    expand.mutate({ action: "expand", current: [], delay: 200 });
    treeProps.clearExpanded();
  }, [scanTask, treeProps.clearExpanded]);
  useEffect(refresh, [refresh]);
  const content = initialLoading ? (
    <Align.Center>
      <Icon.Loading style={{ fontSize: "5rem" }} color="var(--pluto-gray-l5)" />
    </Align.Center>
  ) : expand.isError ? (
    <Status.Text.Centered level="p" shade={6} variant="error">
      Error loading nodes. {expand.error.message}
    </Status.Text.Centered>
  ) : (
    <Tree.Tree loading={loading} {...treeProps} />
  );
  return (
    <Align.Space empty grow>
      <Header.Header level="h4">
        <Header.Title weight={500}>Browser</Header.Title>
        <Header.Actions>
          <Button.Icon onClick={refresh} disabled={scanTask == null || initialLoading}>
            <Icon.Refresh style={{ color: "var(--pluto-gray-l9)" }} />
          </Button.Icon>
        </Header.Actions>
      </Header.Header>
      {content}
    </Align.Space>
  );
};
