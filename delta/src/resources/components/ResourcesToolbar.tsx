// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useState } from "react";

import { OntologyID, OntologyRoot } from "@synnaxlabs/client";
import type { OntologyResource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import type { TreeLeaf } from "@synnaxlabs/pluto";
import { Tree, Space, Client, useAsyncEffect } from "@synnaxlabs/pluto";
import { useStore } from "react-redux";

import { resourceTypes } from "../resources";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { NavDrawerItem, useLayoutPlacer } from "@/layout";
import { WorkspaceState } from "@/workspace";

const updateTreeEntry = (
  data: TreeLeaf[],
  newEntry: Partial<TreeLeaf>,
  key: string
): void =>
  data.forEach((entry, i) => {
    if (entry.key === key) {
      entry.children = entry.children ?? [];
      data[i] = { ...entry, ...newEntry };
    } else if (entry.children != null) {
      updateTreeEntry(entry.children, newEntry, key);
    }
  });

const convertOntologyResources = (resources: OntologyResource[]): TreeLeaf[] => {
  return resources.map(({ id, entity: { name } }) => {
    const { icon, hasChildren } = resourceTypes[id.type];
    return {
      key: id.toString(),
      name,
      icon,
      hasChildren,
      children: [],
    };
  });
};

const ResourcesTree = (): ReactElement => {
  const client = Client.use();
  const [selected, setSelected] = useState<readonly string[]>([]);
  const [data, setData] = useState<TreeLeaf[]>([]);
  const store = useStore();
  const placer = useLayoutPlacer();

  useAsyncEffect(async () => {
    if (client == null) return;
    const resources = await client.ontology.retrieveChildren(OntologyRoot);
    setData(convertOntologyResources(resources));
  }, [client]);

  return (
    <Space empty style={{ height: "100%" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Resources />}>Resources</ToolbarTitle>
      </ToolbarHeader>
      <Tree
        data={data}
        style={{ overflowY: "auto", overflowX: "hidden", height: "100%" }}
        value={selected}
        onChange={(key) => {
          if (key == null) return;
          const id = OntologyID.parseString(key);
          const { onSelect } = resourceTypes[id.type];
          onSelect?.({
            id,
            placer,
            workspace: (store.getState() as { workspace: WorkspaceState }).workspace,
          });
          setSelected([key]);
        }}
        onExpand={(key) => {
          if (client == null || key.length === 0) return;
          void (async () => {
            const resources = await client.ontology.retrieveChildren(
              OntologyID.parseString(key)
            );
            updateTreeEntry(
              data,
              {
                children: convertOntologyResources(resources),
              },
              key
            );
            setData(data.map((d) => ({ ...d })));
          })();
        }}
      />
    </Space>
  );
};

export const ResourcesToolbar: NavDrawerItem = {
  key: "resources",
  icon: <Icon.Resources />,
  content: <ResourcesTree />,
  initialSize: 350,
  minSize: 250,
  maxSize: 650,
};
