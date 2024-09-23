// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { ontology, ranger, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { type Haul, List, Menu as PMenu, Ranger, Text, Tree } from "@synnaxlabs/pluto";
import { CrudeTimeRange, errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Menu } from "@/components/menu";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";
import { createLayout } from "@/range/CreateLayout";
import { overviewLayout } from "@/range/overview/Overview";
import { select, useSelect } from "@/range/selectors";
import { add, remove, rename, setActive, type StoreState } from "@/range/slice";
import {
  addChildRangeMenuItem,
  addToActivePlotMenuItem,
  addToNewPlotMenuItem,
  deleteMenuItem,
  fromClientRange,
  setAsActiveMenuItem,
  viewDetailsMenuItem,
} from "@/range/Toolbar";

const handleSelect: Ontology.HandleSelect = async ({
  selection,
  client,
  store,
  placeLayout,
}): Promise<void> => {
  const ranges = await client.ranges.retrieve(selection.map((s) => s.id.key));
  store.dispatch(add({ ranges: fromClientRange(ranges) }));
  const first = ranges[0];
  placeLayout({ ...overviewLayout, name: first.name, key: first.key });
};

const handleRename: Ontology.HandleTreeRename = {
  eager: ({ store, id, name }) => {
    store.dispatch(rename({ key: id.key, name }));
    store.dispatch(Layout.rename({ key: id.key, name: name }));
  },
  execute: async ({ client, id, name }) => await client.ranges.rename(id.key, name),
  rollback: ({ store, id }, prevName) => {
    store.dispatch(rename({ key: id.key, name: prevName }));
    store.dispatch(Layout.rename({ key: id.key, name: prevName }));
  },
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
    onError: (e, { addStatus }) =>
      addStatus({
        variant: "error",
        message: `Failed to activate range`,
        description: e.message,
      }),
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
    onError: (e, { addStatus }) =>
      addStatus({
        variant: "error",
        message: `Failed to add range to plot`,
        description: e.message,
      }),
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
    onError: (e, { addStatus }) =>
      addStatus({
        variant: "error",
        message: `Failed to add range to plot`,
        description: e.message,
      }),
  }).mutate;

const useViewDetails = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const placer = Layout.usePlacer();
  return ({ selection: { resources } }) =>
    placer({
      ...overviewLayout,
      name: resources[0].name,
      key: resources[0].id.key,
    });
};

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = useConfirmDelete({
    type: "Range",
    description: "Deleting this range will also delete all child ranges.",
  });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({
      state: { nodes, setNodes },
      selection: { resources, nodes: selectedNodes },
      store,
      removeLayout,
    }) => {
      if (!(await confirm(resources))) throw errors.CANCELED;
      const prevNodes = Tree.deepCopy(nodes);
      const minDepth = Math.min(...selectedNodes.map((n) => n.depth));
      const nodesOfMinDepth = selectedNodes.filter((n) => n.depth === minDepth);
      const descendants = Tree.getDescendants(...nodesOfMinDepth).map(
        (n) => new ontology.ID(n.key).key,
      );
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: nodesOfMinDepth.map((n) => n.key),
        }),
      ]);
      const keys = descendants.concat(
        nodesOfMinDepth.map(({ key }) => new ontology.ID(key).key),
      );
      store.dispatch(remove({ keys }));
      removeLayout(...keys);
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
      if (resources.length === 1) message = `Failed to delete ${resources[0].name}`;
      addStatus({
        variant: "error",
        message,
        description: e.message,
      });
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection,
    selection: { resources, nodes },
  } = props;
  const activeRange = useSelect();
  const layout = Layout.useSelectActiveMosaicLayout();
  const handleDelete = useDelete();
  const addToActivePlot = useAddToActivePlot();
  const addToNewPlot = useAddToNewPlot();
  const activate = useActivate();
  const groupFromSelection = Group.useCreateFromSelection();
  const handleLink = Link.useCopyToClipboard();
  const placer = Layout.usePlacer();
  const handleAddChildRange = () =>
    void placer(createLayout({ initial: { parent: resources[0].id.key } }));
  const viewDetails = useViewDetails();
  const handleSelect = {
    delete: () => handleDelete(props),
    rename: () => Tree.startRenaming(nodes[0].key),
    setAsActive: () => activate(props),
    addToActivePlot: () => addToActivePlot(props),
    addToNewPlot: () => addToNewPlot(props),
    group: () => groupFromSelection(props),
    viewDetails: () => viewDetails(props),
    link: () =>
      handleLink({
        name: resources[0].name,
        ontologyID: resources[0].id.payload,
      }),
    addChildRange: handleAddChildRange,
  };
  const isSingle = resources.length === 1;
  return (
    <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      {isSingle && (
        <>
          {resources[0].id.key !== activeRange?.key && setAsActiveMenuItem}
          {viewDetailsMenuItem}
          <PMenu.Divider />
          <Menu.RenameItem />
          {addChildRangeMenuItem}
          <PMenu.Divider />
        </>
      )}
      <Group.GroupMenuItem selection={selection} />
      {layout?.type === "lineplot" && addToActivePlotMenuItem}
      {addToNewPlotMenuItem}
      <PMenu.Divider />
      {deleteMenuItem}
      <PMenu.Divider />
      {isSingle && (
        <>
          <Link.CopyMenuItem />
          <PMenu.Divider />
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const haulItems = ({ id }: ontology.Resource): Haul.Item[] => [
  {
    type: ranger.ONTOLOGY_TYPE,
    key: id.key,
  },
];

const PaletteListItem: Ontology.PaletteListItem = (props) => {
  const { entry } = props;
  return (
    <List.ItemFrame
      direction="y"
      size={0.5}
      style={{ padding: "1.5rem" }}
      highlightHovered
      {...props}
    >
      <Text.WithIcon
        startIcon={<Icon.Range />}
        level="p"
        weight={450}
        shade={9}
        size="medium"
      >
        {entry.name}{" "}
      </Text.WithIcon>
      <Ranger.TimeRangeChip
        level="small"
        timeRange={entry.data?.timeRange as CrudeTimeRange}
      />
    </List.ItemFrame>
  );
};

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
  PaletteListItem,
};
