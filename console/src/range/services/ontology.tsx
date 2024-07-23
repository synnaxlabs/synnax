// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { type ontology, type ranger, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { type Haul, Icon as PIcon, Menu as PMenu, Tree } from "@synnaxlabs/pluto";
import { errors, id, toArray } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components/menu";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";
import { createEditLayout } from "@/range/EditLayout";
import { select, useSelect } from "@/range/selectors";
import {
  add,
  type Range,
  remove,
  rename,
  setActive,
  type StoreState,
} from "@/range/slice";

export const fromClientRange = (ranges: ranger.Range | ranger.Range[]): Range[] =>
  toArray(ranges).map((range) => ({
    variant: "static",
    key: range.key,
    name: range.name,
    timeRange: {
      start: Number(range.timeRange.start.valueOf()),
      end: Number(range.timeRange.end.valueOf()),
    },
    persisted: true,
  }));

const handleSelect: Ontology.HandleSelect = async ({
  selection,
  client,
  store,
}): Promise<void> => {
  const ranges = await client.ranges.retrieve(selection.map((s) => s.id.key));
  store.dispatch(add({ ranges: fromClientRange(ranges) }));
};

const handleRename: Ontology.HandleTreeRename = {
  eager: ({ store, id, name }) => store.dispatch(rename({ key: id.key, name })),
  execute: async ({ client, id, name }) => await client.ranges.rename(id.key, name),
  rollback: ({ store, id }, prevName) =>
    store.dispatch(rename({ key: id.key, name: prevName })),
};

const fetchIfNotInState = async (
  store: Store<StoreState>,
  client: Synnax,
  key: string,
): Promise<void> => {
  const existing = select(store.getState(), key);
  if (existing == null) {
    const range = await client.ranges.retrieve(key);
    store.dispatch(add({ ranges: fromClientRange(range) }));
  }
};

const useActivate = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps>({
    mutationFn: async ({ selection, client, store }) => {
      const res = selection.resources[0];
      await fetchIfNotInState(store, client, res.id.key);
      store.dispatch(setActive(res.id.key));
    },
    onError: (e, { addStatus }) => {
      addStatus({
        key: id.id(),
        variant: "error",
        message: `Failed to activate range`,
        description: e.message,
      });
    },
  }).mutate;

const useAddToActivePlot = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps>({
    mutationFn: async ({ selection, client, store }) => {
      const active = Layout.selectActiveMosaicLayout(store.getState());
      if (active == null) return;
      const res = selection.resources[0];
      await fetchIfNotInState(store, client, res.id.key);
      store.dispatch(
        LinePlot.setRanges({
          key: active.key,
          axisKey: "x1",
          mode: "add",
          ranges: [res.id.key],
        }),
      );
    },
    onError: (e, { addStatus }) => {
      addStatus({
        key: id.id(),
        variant: "error",
        message: `Failed to add range to plot`,
        description: e.message,
      });
    },
  }).mutate;

const useAddToNewPlot = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps>({
    mutationFn: async ({ selection, client, store, placeLayout }) => {
      const res = selection.resources[0];
      await fetchIfNotInState(store, client, res.id.key);
      placeLayout(
        LinePlot.create({
          name: `Plot for ${res.name}`,
          ranges: {
            x1: [res.id.key],
            x2: [],
          },
        }),
      );
    },
    onError: (e, { addStatus }) => {
      addStatus({
        key: id.id(),
        variant: "error",
        message: `Failed to add range to plot`,
        description: e.message,
      });
    },
  }).mutate;

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = useConfirmDelete({ type: "Range" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({
      state: { nodes, setNodes },
      selection: { resources },
      store,
    }) => {
      if (!(await confirm(resources))) throw errors.CANCELED;
      const prevNodes = Tree.deepCopy(nodes);
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: resources.map(({ id }) => id.toString()),
        }),
      ]);
      const keys = resources.map(({ id }) => id.key);
      store.dispatch(remove({ keys }));
      return prevNodes;
    },
    mutationFn: async ({ selection, client }) =>
      await client.ranges.delete(selection.resources.map((r) => r.id.key)),
    onError: (
      e,
      { addStatus, selection: { resources }, state: { setNodes }, store },
      prevNodes,
    ) => {
      if (errors.CANCELED.matches(e)) return;
      if (prevNodes != null) {
        setNodes(prevNodes);
        const ranges = fromClientRange(
          resources.map((resource) => resource.data as unknown as ranger.Range),
        );
        store.dispatch(add({ ranges }));
      }
      let message = "Failed to delete ranges";
      if (resources.length === 1)
        message = `Failed to delete range ${resources[0].name}`;
      addStatus({
        key: id.id(),
        variant: "error",
        message,
        description: e.message,
      });
    },
  }).mutate;
};

const handleEdit = ({
  selection: { resources },
  placeLayout,
}: Ontology.TreeContextMenuProps): void => {
  placeLayout({
    ...createEditLayout("Range.Edit"),
    key: resources[0].id.key,
  });
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection,
    selection: { resources, nodes },
  } = props;
  const activeRange = useSelect();
  const layout = Layout.useSelectActiveMosaicLayout();
  const del = useDelete();
  const addToActivePlot = useAddToActivePlot();
  const addToNewPlot = useAddToNewPlot();
  const activate = useActivate();
  const groupFromSelection = Group.useCreateFromSelection();
  const handleLink = Link.useCopyToClipboard();
  const handleSelect = {
    delete: () => del(props),
    rename: () => Tree.startRenaming(nodes[0].key),
    activate: () => activate(props),
    addToActivePlot: () => addToActivePlot(props),
    addToNewPlot: () => addToNewPlot(props),
    edit: () => handleEdit(props),
    group: () => groupFromSelection(props),
    link: () =>
      handleLink({
        name: resources[0].name,
        resource: resources[0].id.payload,
      }),
  };
  const isSingle = resources.length === 1;
  return (
    <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      <Group.GroupMenuItem selection={selection} />
      {isSingle && (
        <>
          {resources[0].id.key !== activeRange?.key && (
            <PMenu.Item itemKey="activate">Set as Active Range</PMenu.Item>
          )}
          <Menu.RenameItem />
          <PMenu.Item itemKey="edit" startIcon={<Icon.Edit />}>
            Edit
          </PMenu.Item>
        </>
      )}
      <PMenu.Divider />
      {layout?.type === "lineplot" && (
        <PMenu.Item
          itemKey="addToActivePlot"
          startIcon={
            <PIcon.Icon topRight={<Icon.Range />}>
              <Icon.Visualize key="plot" />
            </PIcon.Icon>
          }
        >
          Add to {layout.name}
        </PMenu.Item>
      )}
      <PMenu.Item
        itemKey="addToNewPlot"
        startIcon={
          <PIcon.Create>
            <Icon.Visualize key="plot" />
          </PIcon.Create>
        }
      >
        Add to New Plot
      </PMenu.Item>
      <PMenu.Divider />
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      {isSingle && <Link.CopyMenuItem />}
      <PMenu.Divider />
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const haulItems = ({ id }: ontology.Resource): Haul.Item[] => [
  {
    type: "range",
    key: id.key,
  },
];

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "range",
  hasChildren: true,
  icon: <Icon.Range />,
  canDrop: () => true,
  onSelect: handleSelect,
  TreeContextMenu,
  haulItems,
  allowRename: () => true,
  onRename: handleRename,
};
