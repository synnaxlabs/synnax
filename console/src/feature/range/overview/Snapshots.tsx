// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource, schematic, task } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Schematic, Synnax, Task } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type FC } from "react";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { Tree } from "@/platform/tree";

const SnapshotsListItem = ({ className, ...rest }: List.ItemProps<string>) => {
  const { itemKey } = rest;
  const entry = List.useItem<string, Tree.Resource>(itemKey);
  const services = Range.useSnapshotServices();
  const openTab = Panel.useOpenTab();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const promptConfirm = Modals.useConfirmDelete({ type: "Snapshot" });
  if (entry == null) return null;
  const { id, name } = entry;
  const svc = services[id.type];
  if (svc == null) return null;
  const handleSelect = () => {
    handleError(() => svc.onClick(entry, { client, openTab }), {
      message: "Failed to open snapshot",
      name,
    });
  };
  const handleDelete = () => {
    handleError(
      async () => {
        const confirmed = await promptConfirm({ name });
        if (!confirmed) return;
        await svc.onDelete(entry, { client, openTab });
      },
      { message: "Failed to delete snapshot", name },
    );
  };
  return (
    <List.Item
      className={CSS.cls(CSS.BE("snapshots", "list-item"), className)}
      {...rest}
      justify="between"
      onSelect={handleSelect}
    >
      <Text.Text weight={450}>
        {svc.icon}
        {name}
      </Text.Text>
      <Button.Button
        aria-label={`Delete ${name}`}
        onClick={handleDelete}
        className={CSS.BE("snapshots", "delete")}
        variant="text"
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

/** Shows the records of a list as resources of the type, keyed by resource ID. */
const useResources = <K extends string, E extends record.Keyed<K>>(
  type: resource.Type,
  source: Pick<List.FrameProps<K, E>, "data" | "getItem" | "subscribe">,
  toNamed: (entry: E) => record.KeyedNamed,
) =>
  List.useTransformedData({
    source,
    toKey: (key) => resource.idToString({ type, key }),
    fromKey: (id) => {
      const parsed = resource.parseID(id);
      return parsed.type === type ? (parsed.key as K) : null;
    },
    transform: (entry) => Tree.toResource(type)(toNamed(entry)),
  });

export const Snapshots: FC<SnapshotsProps> = ({ rangeKey }) => {
  const taskList = Task.useList({
    initialQuery: { ranges: [rangeKey], snapshot: true },
  });
  const schematicList = Schematic.useList({
    initialQuery: { ranges: [rangeKey], snapshot: true },
  });
  const tasks = useResources(task.TYPE_RESOURCE_ID.type, taskList, (t) => t.payload);
  const schematics = useResources(
    schematic.TYPE_RESOURCE_ID.type,
    schematicList,
    (s) => s,
  );
  const taskPager = List.usePager({ retrieve: taskList.retrieve });
  const schematicPager = List.usePager({ retrieve: schematicList.retrieve });
  const combined = List.useCombinedData<string, Tree.Resource>({
    first: tasks,
    second: schematics,
  });
  if (taskList.status.variant === "error" || schematicList.status.variant === "error")
    return null;
  const handleFetchMore = () => {
    taskPager.fetchMore();
    schematicPager.fetchMore();
  };
  return (
    <Flex.Box y>
      <Header.Header level="h4" borderColor={6}>
        <Header.Title>Snapshots</Header.Title>
      </Header.Header>
      <List.Frame {...combined} onFetchMore={handleFetchMore}>
        <List.Scroll>
          <List.Items>{snapshotsListItem}</List.Items>
        </List.Scroll>
      </List.Frame>
    </Flex.Box>
  );
};
