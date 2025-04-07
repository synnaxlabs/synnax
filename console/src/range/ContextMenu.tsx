// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { ranger, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Icon as PIcon,
  Menu as PMenu,
  Status,
  Synnax as PSynnax,
  Text,
} from "@synnaxlabs/pluto";
import { errors, toArray } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement } from "react";
import { useDispatch, useStore } from "react-redux";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
import { NULL_CLIENT_ERROR } from "@/errors";
import { Layout } from "@/layout";
import {
  create as createLinePlot,
  LAYOUT_TYPE as LINE_PLOT_LAYOUT_TYPE,
} from "@/lineplot/layout";
import { setRanges as setLinePlotRanges } from "@/lineplot/slice";
import { Link } from "@/link";
import { Modals } from "@/modals";
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
    <PMenu.Item itemKey="rangeSnapshot" startIcon={<Icon.Snapshot />}>
      Snapshot to {range.name}
    </PMenu.Item>
  ) : null;

export const fromClientRange = (ranges: ranger.Range | ranger.Range[]): Range[] =>
  toArray(ranges).map((range) => ({
    variant: "static",
    key: range.key,
    name: range.name,
    timeRange: range.timeRange.numeric,
    persisted: true,
  }));

export const fetchIfNotInState = async (
  store: Store<StoreState>,
  client: Synnax,
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
  const client = PSynnax.use();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return useMutation<void, Error, string>({
    mutationFn: async (key: string) => {
      if (client == null) throw NULL_CLIENT_ERROR;
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
  const client = PSynnax.use();
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
  <PMenu.Item startIcon={<Icon.Delete />} itemKey="delete">
    Delete
  </PMenu.Item>
);

export const setAsActiveMenuItem = (
  <PMenu.Item itemKey="setAsActive" startIcon={<Icon.Dynamic />} iconSpacing="small">
    Set as Active Range
  </PMenu.Item>
);

export const clearActiveMenuItem = (
  <PMenu.Item itemKey="clearActive" startIcon={<Icon.Dynamic />} iconSpacing="small">
    Clear Active Range
  </PMenu.Item>
);

export const viewDetailsMenuItem = (
  <PMenu.Item startIcon={<Icon.Details />} itemKey="details">
    View Details
  </PMenu.Item>
);

export const addToNewPlotMenuItem = (
  <PMenu.Item
    itemKey="addToNewPlot"
    startIcon={
      <PIcon.Create>
        <Icon.LinePlot key="plot" />
      </PIcon.Create>
    }
  >
    Add to New Plot
  </PMenu.Item>
);

export const addToActivePlotMenuItem = (
  <PMenu.Item
    itemKey="addToActivePlot"
    startIcon={
      <PIcon.Icon topRight={<Icon.Range />}>
        <Icon.LinePlot key="plot" />
      </PIcon.Icon>
    }
  >
    Add to Active Plot
  </PMenu.Item>
);

export const addChildRangeMenuItem = (
  <PMenu.Item
    itemKey="addChildRange"
    startIcon={
      <PIcon.Create>
        <Icon.Range />
      </PIcon.Create>
    }
  >
    Create Child Range
  </PMenu.Item>
);

const useViewDetails = (): ((key: string) => void) => {
  const store = useStore<RootState>();
  const client = PSynnax.use();
  const handleError = Status.useErrorHandler();
  const placeLayout = Layout.usePlacer();
  return useMutation<void, Error, string>({
    mutationFn: async (key: string) => {
      if (client == null) throw NULL_CLIENT_ERROR;
      const rng = await fetchIfNotInState(store, client, key);
      placeLayout({ ...OVERVIEW_LAYOUT, name: rng.name, key: rng.key });
    },
    onError: (e) => handleError(e, "Failed to view details"),
  }).mutate;
};

export const ContextMenu = ({ keys: [key] }: PMenu.ContextMenuMenuProps) => {
  const dispatch = useDispatch();
  const client = PSynnax.use();
  const remover = Layout.useRemover();
  const ranges = useSelectMultiple();
  const handleCreate = (key?: string): void => {
    placeLayout(createCreateLayout({ key }));
  };

  const handleRemove = (keys: string[]): void => {
    dispatch(remove({ keys }));
  };
  const handleError = Status.useErrorHandler();

  const confirm = Modals.useConfirm();

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

  const del = useMutation<void, Error, string, Range | undefined>({
    onMutate: async (key: string) => {
      const rng = ranges.find((r) => r.key === key);
      if (
        !(await confirm({
          message: `Are you sure you want to delete ${rng?.name}?`,
          description: "This action cannot be undone.",
          cancel: { label: "Cancel" },
          confirm: { label: "Delete", variant: "error" },
        }))
      )
        throw errors.CANCELED;
      handleRemove([key]);
      remover(key);
      return rng;
    },
    mutationFn: async (key: string) => await client?.ranges.delete(key),
    onError: (e, _, range) => {
      if (errors.CANCELED.matches(e)) return;
      handleError(e, "Failed to delete range");
      dispatch(add({ ranges: [range as Range] }));
    },
  });

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
    create: handleCreate,
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
    <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
      <PMenu.Item startIcon={<Icon.Add />} itemKey="create">
        Create New
      </PMenu.Item>
      {rangeExists && (
        <>
          <PMenu.Divider />
          {rng.key !== activeRange?.key ? setAsActiveMenuItem : clearActiveMenuItem}
          {rng.persisted && viewDetailsMenuItem}
          <PMenu.Divider />
          <Menu.RenameItem />
          {rng.persisted && addChildRangeMenuItem}
          <PMenu.Divider />
          {activeLayout?.type === LINE_PLOT_LAYOUT_TYPE && addToActivePlotMenuItem}
          {addToNewPlotMenuItem}
          <PMenu.Divider />
          <PMenu.Item startIcon={<Icon.Close />} itemKey="remove">
            Remove from List
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
                <PMenu.Item startIcon={<Icon.Save />} itemKey="save">
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
