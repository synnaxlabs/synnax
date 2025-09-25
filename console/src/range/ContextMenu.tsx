// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { DisconnectedError, ranger, type Synnax as Client } from "@synnaxlabs/client";
import { Icon, Menu as PMenu, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { array, errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement } from "react";
import { useDispatch, useStore } from "react-redux";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
import { Layout } from "@/layout";
import {
  create as createLinePlot,
  LAYOUT_TYPE as LINE_PLOT_LAYOUT_TYPE,
} from "@/lineplot/layout";
import { setRanges as setLinePlotRanges } from "@/lineplot/slice";
import { Link } from "@/link";
import { useConfirmDelete } from "@/ontology/hooks";
import { createCreateLayout } from "@/range/Create";
import { OVERVIEW_LAYOUT } from "@/range/external";
import { select, useSelect, useSelectMultiple } from "@/range/selectors";
import { add, type Range, remove, setActive, type StoreState } from "@/range/slice";
import { type RootState } from "@/store";

export interface SnapshotMenuItemProps {
  range?: Range | null;
}

export const SnapshotMenuItem = ({
  range,
}: SnapshotMenuItemProps): ReactElement | null =>
  range?.persisted === true ? (
    <PMenu.Item itemKey="rangeSnapshot">
      <Icon.Snapshot />
      Snapshot to {range.name}
    </PMenu.Item>
  ) : null;

export const fromClientRange = (ranges: ranger.Payload | ranger.Payload[]): Range[] =>
  array.toArray(ranges).map((range) => ({
    variant: "static",
    key: range.key,
    name: range.name,
    timeRange: range.timeRange.numeric,
    persisted: true,
  }));

export const fetchIfNotInState = async (
  store: Store<StoreState>,
  client: Client,
  key: string,
): Promise<Range> => {
  const existing = select(store.getState(), key);
  if (existing == null) {
    const range = fromClientRange(await client.ranges.retrieve(key));
    store.dispatch(add({ ranges: range }));
    return range[0];
  }
  return existing;
};

export const useAddToNewPlot = (): ((key: string) => void) => {
  const store = useStore<RootState>();
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return useMutation<void, Error, string>({
    mutationFn: async (key: string) => {
      if (client == null) throw new DisconnectedError();
      const res = await fetchIfNotInState(store, client, key);
      placeLayout(
        createLinePlot({ name: `Plot for ${res.name}`, ranges: { x1: [key], x2: [] } }),
      );
    },
    onError: (e) => handleError(e, "Failed to add range to new plot"),
  }).mutate;
};

const useAddToActivePlot = (): ((key: string) => void) => {
  const store = useStore<RootState>();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  return useMutation<void, Error, string>({
    mutationFn: async (key: string) => {
      const active = Layout.selectActiveMosaicLayout(store.getState());
      if (active == null || client == null) return;
      await fetchIfNotInState(store, client, key);
      store.dispatch(
        setLinePlotRanges({
          key: active.key,
          axisKey: "x1",
          mode: "add",
          ranges: [key],
        }),
      );
    },
    onError: (e) => handleError(e, "Failed to add range to plot"),
  }).mutate;
};

export const deleteMenuItem = (
  <PMenu.Item itemKey="delete">
    <Icon.Delete />
    Delete
  </PMenu.Item>
);

export const setAsActiveMenuItem = (
  <PMenu.Item itemKey="setAsActive" gap="small">
    <Icon.Dynamic />
    Set as Active Range
  </PMenu.Item>
);

export const clearActiveMenuItem = (
  <PMenu.Item itemKey="clearActive" gap="small">
    <Icon.Dynamic />
    Clear Active Range
  </PMenu.Item>
);

export const viewDetailsMenuItem = (
  <PMenu.Item itemKey="details">
    <Icon.Details />
    View Details
  </PMenu.Item>
);

const AddToNewPlotIcon = Icon.createComposite(Icon.LinePlot, {
  topRight: Icon.Add,
});

export const addToNewPlotMenuItem = (
  <PMenu.Item itemKey="addToNewPlot">
    <AddToNewPlotIcon key="plot" />
    Add to New Plot
  </PMenu.Item>
);

const AddToActivePlotIcon = Icon.createComposite(Icon.LinePlot, {
  topRight: Icon.Range,
});

export const addToActivePlotMenuItem = (
  <PMenu.Item itemKey="addToActivePlot">
    <AddToActivePlotIcon key="plot" />
    Add to Active Plot
  </PMenu.Item>
);

export const CreateChildRangeIcon = Icon.createComposite(Icon.Range, {
  topRight: Icon.Add,
});

export const createChildRangeMenuItem = (
  <PMenu.Item itemKey="addChildRange">
    <CreateChildRangeIcon key="plot" />
    Create Child Range
  </PMenu.Item>
);

export const useViewDetails = (): ((key: string) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const placeLayout = Layout.usePlacer();
  return useMutation<void, Error, string>({
    mutationFn: async (key: string) => {
      if (client == null) throw new DisconnectedError();
      const rng = await client.ranges.retrieve(key);
      placeLayout({ ...OVERVIEW_LAYOUT, name: rng.name, key: rng.key });
    },
    onError: (e) => handleError(e, "Failed to view details"),
  }).mutate;
};

export const useDelete = () => {
  const dispatch = useDispatch();
  const client = Synnax.use();
  const remover = Layout.useRemover();
  const ranges = useSelectMultiple();
  const handleRemove = (keys: string[]): void => {
    dispatch(remove({ keys }));
  };
  const handleError = Status.useErrorHandler();
  const confirm = useConfirmDelete({
    type: "Range",
    description: "Deleting this range will also delete all child ranges.",
  });

  return useMutation<void, Error, string, Range | undefined>({
    onMutate: async (key: string) => {
      const rng = ranges.find((r) => r.key === key);
      if (rng == null || !(await confirm(rng))) throw new errors.Canceled();
      handleRemove([key]);
      remover(key);
      return rng;
    },
    mutationFn: async (key: string) => await client?.ranges.delete(key),
    onError: (e, _, range) => {
      if (errors.Canceled.matches(e)) return;
      handleError(e, "Failed to delete range");
      dispatch(add({ ranges: [range as Range] }));
    },
  });
};

export const ContextMenu = ({ keys: [key] }: PMenu.ContextMenuMenuProps) => {
  const dispatch = useDispatch();
  const client = Synnax.use();
  const ranges = useSelectMultiple();
  const handleCreate = (key?: string): void => {
    placeLayout(createCreateLayout({ key }));
  };

  const handleRemove = (keys: string[]): void => {
    dispatch(remove({ keys }));
  };
  const handleError = Status.useErrorHandler();

  const rng = ranges.find((r) => r.key === key);
  const activeLayout = Layout.useSelectActiveMosaicLayout();
  const addToActivePlot = useAddToActivePlot();
  const addToNewPlot = useAddToNewPlot();
  const activeRange = useSelect();
  const placeLayout = Layout.usePlacer();
  const handleSetActive = () => {
    dispatch(setActive(key));
  };
  const handleClearActive = () => {
    dispatch(setActive(null));
  };
  const handleViewDetails = useViewDetails();
  const handleAddChildRange = () => {
    placeLayout(createCreateLayout({ parent: key }));
  };

  const rangeExists = rng != null;

  const del = useDelete();

  const save = useMutation<void, Error, string, Range | undefined>({
    onMutate: async (key: string) => {
      const range = ranges.find((r) => r.key === key);
      if (range == null || range.variant === "dynamic") return;
      dispatch(add({ ranges: [{ ...range, persisted: true }] }));
      return range;
    },
    mutationFn: async (key: string) => {
      const range = ranges.find((r) => r.key === key);
      if (range == null || range.variant === "dynamic") return;
      await client?.ranges.create({ ...range });
    },
    onError: (e) => handleError(e, "Failed to save range"),
  });

  const handleLink = Cluster.useCopyLinkToClipboard();

  const handleSelect: PMenu.MenuProps["onChange"] = {
    rename: () => Text.edit(`text-${key}`),
    create: () => handleCreate(),
    remove: () => rangeExists && handleRemove([rng.key]),
    delete: () => rangeExists && del.mutate(rng.key),
    details: () => rangeExists && handleViewDetails(rng.key),
    save: () => rangeExists && save.mutate(rng.key),
    link: () =>
      rangeExists &&
      handleLink({ name: rng.name, ontologyID: ranger.ontologyID(rng.key) }),
    addToActivePlot: () => addToActivePlot(key),
    addToNewPlot: () => addToNewPlot(key),
    addChildRange: handleAddChildRange,
    setAsActive: handleSetActive,
    clearActive: handleClearActive,
  };
  return (
    <PMenu.Menu onChange={handleSelect} level="small" gap="small">
      <PMenu.Item itemKey="create">
        <Icon.Add />
        Create New
      </PMenu.Item>
      {rangeExists && (
        <>
          <PMenu.Divider />
          {rng.key !== activeRange?.key ? setAsActiveMenuItem : clearActiveMenuItem}
          {rng.persisted && viewDetailsMenuItem}
          <PMenu.Divider />
          <Menu.RenameItem />
          {rng.persisted && createChildRangeMenuItem}
          <PMenu.Divider />
          {activeLayout?.type === LINE_PLOT_LAYOUT_TYPE && addToActivePlotMenuItem}
          {addToNewPlotMenuItem}
          <PMenu.Divider />
          <PMenu.Item itemKey="remove">
            <Icon.Close />
            Remove from Favorites
          </PMenu.Item>
          {rng.persisted ? (
            <>
              {deleteMenuItem}
              <PMenu.Divider />
              <Link.CopyMenuItem />
            </>
          ) : (
            client != null && (
              <>
                <PMenu.Divider />
                <PMenu.Item itemKey="save">
                  <Icon.Save />
                  Save to Synnax
                </PMenu.Item>
              </>
            )
          )}
        </>
      )}
      <PMenu.Divider />
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};
