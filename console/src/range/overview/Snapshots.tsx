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
  Component,
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

const SnapshotsListItem = (props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const entry = List.useItem<string, ontology.Resource>(itemKey);
  if (entry == null) return null;
  const { id, name } = entry;
  const svc = SNAPSHOTS[id.type as keyof typeof SNAPSHOTS];
  const placeLayout = Layout.usePlacer();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const handleSelect = () => {
    svc
      .onClick(entry, { client, placeLayout })
      .catch((e) => handleError(e, `Failed to open ${entry.name}`));
  };
  return (
    <List.Item
      style={{ padding: "1.5rem" }}
      size="tiny"
      {...props}
      onSelect={handleSelect}
    >
      <Text.WithIcon startIcon={svc.icon} level="p" weight={450} shade={11}>
        {name}
      </Text.WithIcon>
    </List.Item>
  );
};

const snapshotsListItem = Component.renderProp(SnapshotsListItem);

const EMPTY_LIST_CONTENT = (
  <Text.Text level="p" weight={400} shade={10}>
    No Snapshots.
  </Text.Text>
);

export interface SnapshotsProps {
  rangeKey: string;
}

export const Snapshots: FC<SnapshotsProps> = ({ rangeKey }) => {
  const { useListItem, data } = Ontology.useChildren({
    initialParams: { id: ranger.ontologyID(rangeKey) },
    filter: (item) => item.data?.snapshot === true,
  });
  return (
    <Align.Space y>
      <Text.Text level="h4" shade={10} weight={500}>
        Snapshots
      </Text.Text>
      <List.Frame data={data} useListItem={useListItem}>
        <List.Items emptyContent={EMPTY_LIST_CONTENT}>{snapshotsListItem}</List.Items>
      </List.Frame>
    </Align.Space>
  );
};
