// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
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
import { type FC, useCallback } from "react";

import { CSS } from "@/platform/css";
import { Layout } from "@/platform/layout";
import { Range } from "@/platform/range";
import { Tree } from "@/platform/tree";

const SnapshotsListItem = ({ className, ...rest }: List.ItemProps<string>) => {
  const { itemKey } = rest;
  const entry = List.useItem<string, Tree.Entry>(itemKey);
  const services = Range.useSnapshotServices();
  if (entry == null) return null;
  const svc = services[entry.id.type];
  if (svc == null) return null;
  return (
    <SnapshotsListItemContent {...rest} className={className} svc={svc} entry={entry} />
  );
};

interface SnapshotsListItemContentProps extends List.ItemProps<string> {
  svc: Range.SnapshotService;
  entry: Tree.Entry;
}

const SnapshotsListItemContent = ({
  className,
  svc,
  entry,
  ...rest
}: SnapshotsListItemContentProps) => {
  const placeLayout = Layout.usePlacer();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const promptConfirm = Tree.useConfirmDelete({ type: "Snapshot" });
  const { id, name } = entry;
  const isSnapshot = svc.useIsSnapshot(id.key);
  if (!isSnapshot) return null;
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
  const services = Range.useSnapshotServices();
  const { data, getItem, subscribe, retrieve, status } = Ontology.useListChildren({
    initialQuery: { id: ranger.ontologyID(rangeKey) },
    filter: useCallback(
      (item: Tree.Entry) => services[item.id.type] != null,
      [services],
    ),
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
