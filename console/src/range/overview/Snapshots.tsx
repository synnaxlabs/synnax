// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DisconnectedError,
  type ontology,
  ranger,
  schematic,
  type Synnax as Client,
  task,
} from "@synnaxlabs/client";
import {
  Align,
  componentRenderProp,
  Icon,
  List,
  Ontology,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { type FC } from "react";

import { retrieveAndPlaceLayout as retrieveAndPlaceTaskLayout } from "@/hardware/task/layouts";
import { Layout } from "@/layout";
import { create } from "@/schematic/Schematic";

interface SnapshotCtx {
  client: Client | null;
  placeLayout: Layout.Placer;
}

interface SnapshotService {
  icon: Icon.ReactElement;
  onClick: (res: ontology.Resource, ctx: SnapshotCtx) => Promise<void>;
}

const SNAPSHOTS: Record<schematic.OntologyType | task.OntologyType, SnapshotService> = {
  [schematic.ONTOLOGY_TYPE]: {
    icon: <Icon.Schematic />,
    onClick: async ({ id: { key } }, { client, placeLayout }) => {
      if (client == null) throw new DisconnectedError();
      const s = await client.workspaces.schematic.retrieve(key);
      placeLayout(
        create({ ...s.data, key: s.key, name: s.name, snapshot: s.snapshot }),
      );
    },
  },
  [task.ONTOLOGY_TYPE]: {
    icon: <Icon.Task />,
    onClick: async ({ id: { key } }, { client, placeLayout }) =>
      retrieveAndPlaceTaskLayout(client, key, placeLayout),
  },
};

const SnapshotsListItem = (props: List.ItemProps<string, ontology.Resource>) => {
  const { entry } = props;
  const { id, name } = entry;
  const svc = SNAPSHOTS[id.type as keyof typeof SNAPSHOTS];
  const placeLayout = Layout.usePlacer();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const handleSelect = () => {
    handleError(
      svc.onClick(entry, { client, placeLayout }),
      `Failed to open ${entry.name}`,
    );
  };
  return (
    <List.ItemFrame
      style={{ padding: "1.5rem" }}
      gap="tiny"
      {...props}
      onSelect={handleSelect}
    >
      <Text.WithIcon startIcon={svc.icon} level="p" weight={450} shade={11}>
        {name}
      </Text.WithIcon>
    </List.ItemFrame>
  );
};

const snapshotsListItem = componentRenderProp(SnapshotsListItem);

const EMPTY_LIST_CONTENT = (
  <Text.Text level="p" weight={400} shade={10}>
    No Snapshots.
  </Text.Text>
);

export interface SnapshotsProps {
  rangeKey: string;
}

export const Snapshots: FC<SnapshotsProps> = ({ rangeKey }) => {
  const { data: snapshots } = Ontology.useChildren(ranger.ontologyID(rangeKey));
  if (snapshots == null) return null;
  const filtered = snapshots.filter(({ data }) => data?.snapshot === true);
  return (
    <Align.Space y>
      <Text.Text level="h4" shade={10} weight={500}>
        Snapshots
      </Text.Text>
      <List.List data={filtered} emptyContent={EMPTY_LIST_CONTENT}>
        <List.Core empty>{snapshotsListItem}</List.Core>
      </List.List>
    </Align.Space>
  );
};
