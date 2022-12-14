// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { OntologyID, OntologyRoot } from "@synnaxlabs/client";
import type { OntologyResource } from "@synnaxlabs/client";
import { Tree, Header, Space } from "@synnaxlabs/pluto";
import type { TreeLeaf } from "@synnaxlabs/pluto";
import { AiFillFolder } from "react-icons/ai";
import { useStore } from "react-redux";

import { resourceTypes } from "../resources";

import { useClusterClient } from "@/features/cluster";
import { useLayoutPlacer } from "@/features/layout";
import { WorkspaceState } from "@/features/workspace";
import { useAsyncEffect } from "@/hooks";

const updateTreeEntry = (
  data: TreeLeaf[],
  newEntry: Partial<TreeLeaf>,
  key: string
): void =>
  data.forEach((entry, i) => {
    if (entry.key === key)
      data[i] = { ...entry, ...newEntry, children: entry.children ?? [] };
    else if (entry.children != null) updateTreeEntry(entry.children, newEntry, key);
  });

const convertOntologyResources = (resources: OntologyResource[]): TreeLeaf[] => {
  return resources.map(({ id, entity: { name } }) => {
    const { icon, hasChildren } = resourceTypes[id.type];
    return {
      key: id.toString(),
      title: name,
      icon,
      hasChildren,
      children: [],
    };
  });
};

const ResourcesTree = (): JSX.Element => {
  const client = useClusterClient();
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
      <Header level="h4" divided icon={<AiFillFolder />}>
        Resources
      </Header>
      <Tree
        data={data}
        style={{ overflowY: "auto", overflowX: "hidden" }}
        onSelect={([key]: string[]) => {
          const id = OntologyID.parseString(key);
          const { onSelect } = resourceTypes[id.type];
          onSelect?.({
            id,
            placer,
            workspace: (store.getState() as { workspace: WorkspaceState }).workspace,
          });
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

export const ResourcesToolbar = {
  key: "resources",
  icon: <AiFillFolder />,
  content: <ResourcesTree />,
};
