// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, ranger } from "@synnaxlabs/client";
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

import { CSS } from "@/platform/css";
import { Ontology as ServiceOntology } from "@/platform/ontology";
import { Range } from "@/platform/range";
import { Session } from "@/session";

const SnapshotsListItem = ({ className, ...rest }: List.ItemProps<string>) => {
  const { itemKey } = rest;
  const entry = List.useItem<string, ontology.Resource>(itemKey);
  const services = Range.useSnapshotServices();
  const placeLayout = Session.Layout.usePlacer();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const promptConfirm = ServiceOntology.useConfirmDelete({ type: "Snapshot" });
  if (entry == null) return null;
  const { id, name } = entry;
  const svc = services[id.type];
  if (svc == null) return null;
  const handleSelect = () => {
    handleError(
      svc.onClick(entry, { client, placeLayout }),
      `Failed to open ${entry.name}`,
    );
  };
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
