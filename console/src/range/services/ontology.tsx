// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { ontology, type ranger, type Synnax } from "@synnaxlabs/client";
import {
  type Haul,
  Icon,
  List,
  Menu as PMenu,
  Ranger,
  Select,
  Text,
  Tree,
} from "@synnaxlabs/pluto";
import { array, type CrudeTimeRange, errors, strings } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";
import {
  addToActivePlotMenuItem,
  addToNewPlotMenuItem,
  clearActiveMenuItem,
  createChildRangeMenuItem,
  deleteMenuItem,
  fromClientRange,
  setAsActiveMenuItem,
  viewDetailsMenuItem,
} from "@/range/ContextMenu";
import { createCreateLayout } from "@/range/Create";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";
import { select, useSelect } from "@/range/selectors";
import { add, remove, rename, setActive, type StoreState } from "@/range/slice";

const handleSelect: Ontology.HandleSelect = ({
  selection,
  client,
  store,
  placeLayout,
  handleError,
}) => {
  client.ranges
    .retrieve(selection.map((s) => s.id.key))
    .then((ranges) => {
      store.dispatch(add({ ranges: fromClientRange(ranges) }));
      const first = ranges[0];
      placeLayout({ ...OVERVIEW_LAYOUT, name: first.name, key: first.key });
    })
    .catch((e) => {
      const names = strings.naturalLanguageJoin(
        selection.map(({ name }) => name),
        "range",
      );
      handleError(e, `Failed to select ${names}`);
    });
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
  const keyList = array.toArray(keys);
  const missing = keyList.filter((key) => select(store.getState(), key) == null);
  if (missing.length === 0) return;
  const ranges = await client.ranges.retrieve(missing);
  store.dispatch(add({ ranges: fromClientRange(ranges) }));
};

const useActivate = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps>({
    mutationFn: async ({ selection, client, store }) => {
      const id = selection.resourceIDs[0];
      await fetchIfNotInState(store, client, id.key);
      store.dispatch(setActive(id.key));
    },
    onError: (e, { handleError }) => handleError(e, "Failed to activate range"),
  }).mutate;

const useAddToActivePlot = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps>({
    mutationFn: async ({ selection: { resourceIDs }, client, store }) => {
      const active = Layout.selectActiveMosaicLayout(store.getState());
      if (active == null) return;
      const keys = resourceIDs.map((r) => r.key);
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
    onError: (
      e,
      { handleError, selection: { resourceIDs }, state: { getResource } },
    ) => {
      const rangeNames = resourceIDs.map((r) => getResource(r).name);
      handleError(
        e,
        `Failed to add ${strings.naturalLanguageJoin(rangeNames, "range")} to the active plot`,
      );
    },
  }).mutate;

const useAddToNewPlot = (): ((props: Ontology.TreeContextMenuProps) => void) =>
  useMutation<void, Error, Ontology.TreeContextMenuProps>({
    mutationFn: async ({
      selection: { resourceIDs },
      state: { getResource },
      client,
      store,
      placeLayout,
    }) => {
      const keys = resourceIDs.map((r) => r.key);
      await fetchIfNotInState(store, client, keys);
      const names = resourceIDs.map((r) => getResource(r).name);
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
    onError: (
      e,
      { handleError, selection: { resourceIDs }, state: { getResource } },
    ) => {
      const names = resourceIDs.map((r) => getResource(r).name);
      handleError(
        e,
        `Failed to add ${strings.naturalLanguageJoin(names, "range")} to plot`,
      );
    },
  }).mutate;

const useViewDetails = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const placeLayout = Layout.usePlacer();
  return ({ selection: { resourceIDs }, state: { getResource } }) =>
    placeLayout({
      ...OVERVIEW_LAYOUT,
      name: getResource(resourceIDs[0]).name,
      key: resourceIDs[0].key,
    });
};

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = useConfirmDelete({
    type: "Range",
    description: "Deleting this range will also delete all child ranges.",
  });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({
      state: { nodes, setNodes, getResource },
      selection: { resourceIDs },
      store,
      removeLayout,
    }) => {
      const resources = resourceIDs.map((r) => getResource(r));
      const stringIDs = resourceIDs.map((id) => ontology.idToString(id));
      if (!(await confirm(resources))) throw new errors.Canceled();
      setNodes([...Tree.removeNode({ tree: nodes, keys: stringIDs })]);
      const rangesToRemove = resourceIDs
        .filter((id) => id.type === "range")
        .map((id) => id.key);
      store.dispatch(remove({ keys: rangesToRemove }));
      removeLayout(...resourceIDs.map((id) => ontology.idToString(id)));
      return nodes;
    },
    mutationFn: async ({ selection: { resourceIDs }, client }) =>
      await client.ranges.delete(resourceIDs.map((r) => r.key)),
    onError: (
      e,
      {
        handleError,
        selection: { resourceIDs },
        state: { setNodes, getResource },
        store,
      },
      prevNodes,
    ) => {
      if (errors.Canceled.matches(e)) return;
      if (prevNodes != null) {
        setNodes(prevNodes);
        const ranges = fromClientRange(
          resourceIDs.map((id) => getResource(id).data as unknown as ranger.Range),
        );
        store.dispatch(add({ ranges }));
      }
      let message = "Failed to delete ranges";
      if (resourceIDs.length === 1)
        message = `Failed to delete ${getResource(resourceIDs[0]).name}`;
      handleError(e, message);
    },
  }).mutate;
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { resourceIDs, rootID },
    store,
    state: { getResource, shape },
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
  const firstID = resourceIDs[0];
  const firstResource = getResource(firstID);
  const groupFromSelection = Group.useCreateFromSelection();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleAddChildRange = () => {
    placeLayout(createCreateLayout({ parent: firstID.key }));
  };
  const viewDetails = useViewDetails();
  const handleSelect = {
    delete: () => handleDelete(props),
    rename: () => Text.edit(ontology.idToString(resourceIDs[0])),
    setAsActive: () => activate(props),
    addToActivePlot: () => addToActivePlot(props),
    addToNewPlot: () => addToNewPlot(props),
    group: () => groupFromSelection(props),
    details: () => viewDetails(props),
    link: () => handleLink({ name: firstResource.name, ontologyID: firstID }),
    addChildRange: handleAddChildRange,
    clearActive: clearActiveRange,
  };
  const isSingle = resourceIDs.length === 1;
  let showAddToActivePlot = false;
  if (layout?.type === LinePlot.LAYOUT_TYPE) {
    const activeRanges = LinePlot.selectRanges(store.getState(), layout.key).x1.map(
      (r) => r.key,
    );
    showAddToActivePlot = resourceIDs.some((r) => !activeRanges.includes(r.key));
  }

  return (
    <PMenu.Menu onChange={handleSelect} level="small" gap="small">
      {isSingle && (
        <>
          {firstID.key !== activeRange?.key ? setAsActiveMenuItem : clearActiveMenuItem}
          {viewDetailsMenuItem}
          <PMenu.Divider />
          <Menu.RenameItem />
          {createChildRangeMenuItem}
          <PMenu.Divider />
        </>
      )}
      <Group.MenuItem resourceIDs={resourceIDs} shape={shape} rootID={rootID} />
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
  { type: "range", key: id.key },
];

const PaletteListItem: Ontology.PaletteListItem = (props) => {
  const resource = List.useItem<string, ontology.Resource>(props.itemKey);
  return (
    <Select.ListItem gap="tiny" highlightHovered justify="between" {...props}>
      <Text.Text weight={450} gap="medium">
        <Icon.Range />
        {resource?.name}
      </Text.Text>
      <Ranger.TimeRangeChip
        level="small"
        timeRange={resource?.data?.timeRange as CrudeTimeRange}
      />
    </Select.ListItem>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "range",
  icon: <Icon.Range />,
  onSelect: handleSelect,
  canDrop: () => true,
  haulItems,
  allowRename: () => true,
  onRename: handleRename,
  TreeContextMenu,
  PaletteListItem,
};
