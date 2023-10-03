// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type Synnax, type ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu, Tree } from "@synnaxlabs/pluto";
import { toArray } from "@synnaxlabs/x";

import { setActive } from "@/cluster/slice";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { setRanges } from "@/lineplot/slice";
import { Ontology } from "@/ontology";
import { defineWindowLayout } from "@/range/Define";
import { type Range } from "@/range/range";
import { select } from "@/range/selectors";
import { type StoreState, add, remove } from "@/range/slice";

const fromClientRange = (ranges: ranger.Range | ranger.Range[]): Range[] =>
  toArray(ranges).map((range) => ({
    variant: "static",
    key: range.key,
    name: range.name,
    timeRange: {
      start: range.timeRange.start.valueOf(),
      end: range.timeRange.end.valueOf(),
    },
    persisted: true,
  }));

const handleSelect: Ontology.HandleSelect = ({ selection, client, store }) => {
  void (async () => {
    const ranges = await client.ranges.retrieve(selection.map((s) => s.id.key));
    store.dispatch(add({ ranges: fromClientRange(ranges) }));
  })();
};

const handleDelete = async ({
  client,
  store,
  selection: { resources },
  state: { nodes, setNodes },
}: Ontology.TreeContextMenuProps): Promise<void> => {
  const keys = resources.map((r) => r.id.key);
  const ids = resources.map((r) => r.id.toString());
  await client.ranges.delete(keys);
  const next = Tree.removeNode(nodes, ...ids);
  setNodes([...next]);
  store.dispatch(remove({ keys }));
};

const fetchIfNotInState = async (
  store: Store<StoreState>,
  client: Synnax,
  key: string
): Promise<void> => {
  const existing = select(store.getState(), key);
  if (existing == null) {
    const range = await client.ranges.retrieve(key);
    store.dispatch(add({ ranges: fromClientRange(range) }));
  }
};

const handleActivate = async ({
  client,
  store,
  selection: { resources },
}: Ontology.TreeContextMenuProps): Promise<void> => {
  const res = resources[0];
  await fetchIfNotInState(store, client, res.key);
  store.dispatch(setActive(res.key));
};

const handleAddToActivePlot = async ({
  client,
  selection: { resources },
  store,
}: Ontology.TreeContextMenuProps): Promise<void> => {
  const active = Layout.selectActiveMosaicTab(store.getState());
  if (active == null) return;
  const res = resources[0];
  await fetchIfNotInState(store, client, res.key);
  store.dispatch(
    setRanges({
      key: active.key,
      axisKey: "x1",
      mode: "add",
      ranges: [res.key],
    })
  );
};

const handleAddToNewPlot = async ({
  client,
  placeLayout,
  selection: { resources },
  store,
}: Ontology.TreeContextMenuProps): Promise<void> => {
  const res = resources[0];
  await fetchIfNotInState(store, client, res.key);
  placeLayout(
    LinePlot.create({
      name: `Plot for ${res.name}`,
      ranges: {
        x1: [res.key],
        x2: [],
      },
    })
  );
};

const handleEdit = ({
  selection: { resources },
  placeLayout,
}: Ontology.TreeContextMenuProps) => {
  placeLayout({ ...defineWindowLayout, key: resources[0].id.key });
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const { selection, store } = props;
  const state = store.getState();
  const activeRange = select(state);
  const layout = Layout.selectActiveMosaicTab(state);
  const { resources, nodes } = selection;

  const handleSelect = (itemKey: string): void => {
    switch (itemKey) {
      case "delete":
        void handleDelete(props);
        return;
      case "rename":
        Tree.startRenaming(nodes[0].key);
        return;
      case "activate":
        void handleActivate(props);
        return;
      case "addToActivePlot":
        void handleAddToActivePlot(props);
        return;
      case "addToNewPlot":
        void handleAddToNewPlot(props);
        return;
      case "edit":
        handleEdit(props);
    }
  };

  return (
    <Menu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      <Group.GroupMenuItem selection={selection} />
      {resources.length === 1 && (
        <>
          {resources[0].id.key !== activeRange?.key && (
            <Menu.Item itemKey="activate">Set as Active Range</Menu.Item>
          )}
          <Ontology.RenameMenuItem />
          <Menu.Item itemKey="edit" startIcon={<Icon.Edit />}>
            Edit
          </Menu.Item>
        </>
      )}
      {layout?.type === "lineplot" && (
        <Menu.Item itemKey="addToActivePlot" startIcon={<Icon.Visualize />}>
          Add to {layout.name}
        </Menu.Item>
      )}
      <Menu.Item itemKey="addToNewPlot" startIcon={<Icon.Visualize />}>
        Add to New Plot
      </Menu.Item>
      <Menu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </Menu.Item>
    </Menu.Menu>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "range",
  hasChildren: false,
  icon: <Icon.Range />,
  canDrop: () => true,
  onSelect: handleSelect,
  TreeContextMenu,
  haulItems: () => [],
  allowRename: () => true,
};
