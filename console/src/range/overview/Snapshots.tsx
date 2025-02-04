// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, ranger, type Synnax as Client } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  componentRenderProp,
  type Icon as PIcon,
  List,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { type FC, useState } from "react";

import { Task } from "@/hardware/task";
import { Layout } from "@/layout";
import { create } from "@/schematic/external";

interface SnapshotService {
  icon: PIcon.Element;
  onClick: (client: Client, res: ontology.Resource, place: Layout.Placer) => void;
}

const SNAPSHOTS: Record<"schematic" | "task", SnapshotService> = {
  schematic: {
    icon: <Icon.Schematic />,
    onClick: (client, res, place) => {
      void (async () => {
        const s = await client.workspaces.schematic.retrieve(res.id.key);
        place(create({ ...s.data, key: s.key, name: s.name, snapshot: s.snapshot }));
      })();
    },
  },
  task: {
    icon: <Icon.Task />,
    onClick: (client, res, place) =>
      void Task.retrieveAndPlaceLayout(client, res.id.key, place),
  },
};

const SnapshotsListItem = (props: List.ItemProps<string, ontology.Resource>) => {
  const { entry } = props;
  const { id, name } = entry;
  const svc = SNAPSHOTS[id.type as keyof typeof SNAPSHOTS];
  const placeLayout = Layout.usePlacer();
  const client = Synnax.use();
  const handleSelect = () => {
    if (client == null) return;
    svc.onClick(client, entry, placeLayout);
  };
  return (
    <List.ItemFrame
      style={{ padding: "1.5rem" }}
      size={0.5}
      {...props}
      onSelect={handleSelect}
    >
      <Text.WithIcon startIcon={svc.icon} level="p" weight={450} shade={9}>
        {name}
      </Text.WithIcon>
    </List.ItemFrame>
  );
};

const snapshotsListItem = componentRenderProp(SnapshotsListItem);

const EMPTY_LIST_CONTENT = (
  <Text.Text level="p" weight={400} shade={6}>
    No Snapshots.
  </Text.Text>
);

export interface SnapshotsProps {
  rangeKey: string;
}

export const Snapshots: FC<SnapshotsProps> = ({ rangeKey }) => {
  const client = Synnax.use();
  const [snapshots, setSnapshots] = useState<ontology.Resource[]>([]);

  useAsyncEffect(async () => {
    if (client == null) return;
    const otgID = ranger.ontologyID(rangeKey);
    const children = await client.ontology.retrieveChildren(otgID);
    const relevant = children.filter((child) => child.data?.snapshot === true);
    setSnapshots(relevant);
    const tracker = await client.ontology.openDependentTracker({
      target: otgID,
      dependents: relevant,
      relationshipDirection: "from",
    });
    tracker.onChange((snapshots) => {
      const relevant = snapshots.filter((child) => child.data?.snapshot === true);
      setSnapshots(relevant);
    });
    return async () => await tracker.close();
  }, [client, rangeKey]);

  return (
    <Align.Space direction="y">
      <Text.Text level="h4" shade={8} weight={500}>
        Snapshots
      </Text.Text>
      <List.List data={snapshots} emptyContent={EMPTY_LIST_CONTENT}>
        <List.Core empty>{snapshotsListItem}</List.Core>
      </List.List>
    </Align.Space>
  );
};
