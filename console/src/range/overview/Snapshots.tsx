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
  type Synnax as Client,
} from "@synnaxlabs/client";
import {
  Button,
  Component,
  Flex,
  Header,
  Icon,
  List,
  Ontology,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { type FC } from "react";

import { CSS } from "@/css";
import { retrieveAndPlaceLayout as retrieveAndPlaceTaskLayout } from "@/hardware/task/layouts";
import { Layout } from "@/layout";
import { useConfirmDelete } from "@/ontology/hooks";
import { create } from "@/schematic/Schematic";

interface SnapshotCtx {
  client: Client | null;
  placeLayout: Layout.Placer;
}

interface SnapshotService {
  icon: Icon.ReactElement;
  onClick: (res: ontology.Resource, ctx: SnapshotCtx) => Promise<void>;
  onDelete: (res: ontology.Resource, ctx: SnapshotCtx) => Promise<void>;
}

const SNAPSHOTS: Record<"schematic" | "task", SnapshotService> = {
  schematic: {
    icon: <Icon.Schematic />,
    onClick: async ({ id: { key } }, { client, placeLayout }) => {
      if (client == null) throw new DisconnectedError();
      const s = await client.workspaces.schematics.retrieve({ key });
      placeLayout(
        create({ ...s.data, key: s.key, name: s.name, snapshot: s.snapshot }),
      );
    },
    onDelete: async ({ id: { key } }, { client }) => {
      if (client == null) throw new DisconnectedError();
      await client.workspaces.schematics.delete(key);
    },
  },
  task: {
    icon: <Icon.Task />,
    onClick: async ({ id: { key } }, { client, placeLayout }) =>
      retrieveAndPlaceTaskLayout(client, key, placeLayout),
    onDelete: async ({ id: { key } }, { client }) => {
      if (client == null) throw new DisconnectedError();
      await client.tasks.delete(key);
    },
  },
};

const SnapshotsListItem = ({ className, ...rest }: List.ItemProps<string>) => {
  const { itemKey } = rest;
  const entry = List.useItem<string, ontology.Resource>(itemKey);
  if (entry == null) return null;
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
  const promptConfirm = useConfirmDelete({
    type: "Snapshot",
  });
  const handleDelete = () => {
    handleError(async () => {
      const confirmed = await promptConfirm({ name });
      if (!confirmed) return;
      await svc.onDelete(entry, { client, placeLayout });
    }, `Failed to delete ${name}`);
  };
  return (
    <List.Item
      className={CSS(CSS.BE("snapshots", "list-item"), className)}
      {...rest}
      justify="between"
      onSelect={handleSelect}
    >
      <Text.Text weight={450}>
        {svc.icon}
        {name}
      </Text.Text>
      <Button.Button
        onClick={handleDelete}
        className={CSS.BE("snapshots", "delete")}
        variant="shadow"
      >
        <Icon.Delete color={10} />
      </Button.Button>
    </List.Item>
  );
};

const snapshotsListItem = Component.renderProp(SnapshotsListItem);

export interface SnapshotsProps {
  rangeKey: string;
}

export const Snapshots: FC<SnapshotsProps> = ({ rangeKey }) => {
  const { data, getItem, subscribe, retrieve, status } = Ontology.useListChildren({
    initialQuery: { id: ranger.ontologyID(rangeKey) },
    filter: (item) => item.data?.snapshot === true,
  });
  const { fetchMore } = List.usePager({ retrieve });
  if (status.variant === "error") return null;
  return (
    <Flex.Box y>
      <Header.Header level="h4" borderColor={5}>
        <Header.Title>Snapshots</Header.Title>
      </Header.Header>
      <List.Frame
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={fetchMore}
      >
        <List.Items>{snapshotsListItem}</List.Items>
      </List.Frame>
    </Flex.Box>
  );
};
