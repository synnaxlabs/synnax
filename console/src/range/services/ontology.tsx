// Copyright 2025 Synnax Labs, Inc.
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
import { type CrudeTimeRange, errors, strings, toArray } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useDispatch } from "react-redux";

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
  clearActiveMenuItem,
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
    store.dispatch(Layout.rename({ key: id.key, name }));
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
  keys: string | string[],
): Promise<void> => {
  const keyList = toArray(keys);
  const missing = keyList.filter((key) => select(store.getState(), key) == null);
  if (missing.length === 0) return;
  const ranges = await client.ranges.retrieve(missing);
  store.dispatch(add({ ranges: fromClientRange(ranges) }));
};

const useActivate = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps>({
    mutationFn: async ({ selection, client, store }) => {
      const res = selection.resources[0];
      await fetchIfNotInState(store, client, res.id.key);
      store.dispatch(setActive(res.id.key));
    },
    onError: (e, { handleException }) => handleException(e, "Failed to activate range"),
  }).mutate;

const useAddToActivePlot = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps>({
    mutationFn: async ({ selection: { resources }, client, store }) => {
      const active = Layout.selectActiveMosaicLayout(store.getState());
      if (active == null) return;
      const keys = resources.map((r) => r.id.key);
      await fetchIfNotInState(store, client, keys);
      store.dispatch(
        LinePlot.setRanges({
          key: active.key,
          axisKey: "x1",
          mode: "add",
          ranges: keys,
        }),
      );
    },
    onError: (e, { handleException, selection: { resources } }) => {
      const rangeNames = resources.map((r) => r.name);
      handleException(
        e,
        `Failed to add ${strings.naturalLanguageJoin(rangeNames, "range")} to the active plot`,
      );
    },
  }).mutate;

const useAddToNewPlot = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps>({
    mutationFn: async ({ selection: { resources }, client, store, placeLayout }) => {
      const keys = resources.map((r) => r.id.key);
      await fetchIfNotInState(store, client, keys);
      const names = resources.map((r) => r.name);
      placeLayout(
        LinePlot.create({
          name: `Plot for ${strings.naturalLanguageJoin(names, "range")}`,
          ranges: {
            x1: keys,
            x2: [],
          },
        }),
      );
    },
    onError: (e, { handleException, selection: { resources } }) => {
      const names = resources.map((r) => r.name);
      handleException(
        e,
        `Failed to add ${strings.naturalLanguageJoin(names, "range")} to plot`,
      );
    },
  }).mutate;

const useViewDetails = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const place = Layout.usePlacer();
  return ({ selection: { resources } }) =>
    place({
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
      { handleException, selection: { resources }, state: { setNodes }, store },
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
      handleException(e, message);
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection,
    selection: { resources, nodes },
    store,
    placeLayout,
  } = props;
  const activeRange = useSelect();
  const layout = Layout.useSelectActiveMosaicLayout();
  const handleDelete = useDelete();
  const addToActivePlot = useAddToActivePlot();
  const addToNewPlot = useAddToNewPlot();
  const activate = useActivate();
  const dispatch = useDispatch();
  const clearActiveRange = () => {
    dispatch(setActive(null));
  };
  const groupFromSelection = Group.useCreateFromSelection();
  const handleLink = Link.useCopyToClipboard();
  const handleAddChildRange = () => {
    placeLayout(createLayout({ initial: { parent: resources[0].id.key } }));
  };
  const viewDetails = useViewDetails();
  const handleSelect = {
    delete: () => handleDelete(props),
    rename: () => Tree.startRenaming(nodes[0].key),
    setAsActive: () => activate(props),
    addToActivePlot: () => addToActivePlot(props),
    addToNewPlot: () => addToNewPlot(props),
    group: () => groupFromSelection(props),
    details: () => viewDetails(props),
    link: () =>
      handleLink({ name: resources[0].name, ontologyID: resources[0].id.payload }),
    addChildRange: handleAddChildRange,
    clearActive: clearActiveRange,
  };
  const isSingle = resources.length === 1;
  let showAddToActivePlot = false;
  if (layout?.type === LinePlot.LAYOUT_TYPE) {
    const activeRanges = LinePlot.selectRanges(store.getState(), layout.key).x1.map(
      (r) => r.key,
    );
    showAddToActivePlot = resources.some((r) => !activeRanges.includes(r.id.key));
  }

  return (
    <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      {isSingle && (
        <>
          {resources[0].id.key !== activeRange?.key
            ? setAsActiveMenuItem
            : clearActiveMenuItem}
          {viewDetailsMenuItem}
          <PMenu.Divider />
          <Menu.RenameItem />
          {addChildRangeMenuItem}
          <PMenu.Divider />
        </>
      )}
      <Group.GroupMenuItem selection={selection} />
      {showAddToActivePlot && addToActivePlotMenuItem}
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
  { type: ranger.ONTOLOGY_TYPE, key: id.key },
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
  ...Ontology.BASE_SERVICE,
  type: ranger.ONTOLOGY_TYPE,
  icon: <Icon.Range />,
  onSelect: handleSelect,
  canDrop: () => true,
  haulItems,
  allowRename: () => true,
  onRename: handleRename,
  TreeContextMenu,
  PaletteListItem,
};
