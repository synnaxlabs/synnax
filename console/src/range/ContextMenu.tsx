// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { ranger, type Synnax as Client } from "@synnaxlabs/client";
import {
  type Flux,
  Icon,
  Menu as PMenu,
  Ranger,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
import { Layout } from "@/layout";
import { LAYOUT_TYPE as LINE_PLOT_LAYOUT_TYPE } from "@/lineplot/layout";
import { Link } from "@/link";
import { useConfirmDelete } from "@/ontology/hooks";
import { createCreateLayout } from "@/range/Create";
import { OVERVIEW_LAYOUT } from "@/range/external";
import { select, useSelect, useSelectMultiple } from "@/range/selectors";
import { add, type Range, remove, setActive, type StoreState } from "@/range/slice";
import { fromClientRange } from "@/range/translate";
import { useAddToActivePlot } from "@/range/useAddToActivePlot";
import { useAddToNewPlot } from "@/range/useAddToNewPlot";

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

export const addChildRangeMenuItem = (
  <PMenu.Item itemKey="addChildRange">
    <CreateChildRangeIcon key="plot" />
    Create Child Range
  </PMenu.Item>
);

export const useViewDetails = (): ((key: string) => void) => {
  const addStatus = Status.useAdder();
  const placeLayout = Layout.usePlacer();
  const { retrieve } = Ranger.useRetrieveObservable({
    onChange: ({ data, variant, status }) => {
      if (variant !== "success") {
        if (variant === "error") addStatus(status);
        return;
      }
      placeLayout({ ...OVERVIEW_LAYOUT, name: data.name, key: data.key });
    },
  });
  return useCallback((key: string) => retrieve({ key }), [retrieve]);
};

export const useDelete = () => {
  const dispatch = useDispatch();
  const remover = Layout.useRemover();
  const ranges = useSelectMultiple();
  const handleRemove = (keys: string[]): void => {
    dispatch(remove({ keys }));
  };
  const confirm = useConfirmDelete({
    type: "Range",
    description: "Deleting this range will also delete all child ranges.",
  });
  const { update } = Ranger.useDelete({
    beforeUpdate: useCallback(
      async ({ value }: Flux.BeforeUpdateArgs<ranger.Key | ranger.Key[]>) => {
        const keys = array.toArray(value);
        const rng = ranges.filter((r) => keys.includes(r.key));
        if (!(await confirm(rng))) return false;
        handleRemove(keys);
        remover(...keys);
        return true;
      },
      [],
    ),
  });
  return update;
};

export const usePersist = () => {
  const dispatch = useDispatch();
  const ranges = useSelectMultiple();
  const { update } = Ranger.useCreate();
  return useCallback(
    (key: string) => {
      const range = ranges.find((r) => r.key === key);
      if (range == null || range.variant === "dynamic") return;
      dispatch(add({ ranges: [{ ...range, persisted: true }] }));
      update(range);
    },
    [dispatch, ranges],
  );
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
  const persist = usePersist();
  const handleLink = Cluster.useCopyLinkToClipboard();

  const handleSelect: PMenu.MenuProps["onChange"] = {
    rename: () => Text.edit(`text-${key}`),
    create: () => handleCreate(),
    remove: () => rangeExists && handleRemove([rng.key]),
    delete: () => rangeExists && del(rng.key),
    details: () => rangeExists && handleViewDetails(rng.key),
    save: () => rangeExists && persist(rng.key),
    link: () =>
      rangeExists &&
      handleLink({ name: rng.name, ontologyID: ranger.ontologyID(rng.key) }),
    addToActivePlot: () => addToActivePlot([key]),
    addToNewPlot: () => addToNewPlot([key]),
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
          {rng.persisted && addChildRangeMenuItem}
          <PMenu.Divider />
          {activeLayout?.type === LINE_PLOT_LAYOUT_TYPE && addToActivePlotMenuItem}
          {addToNewPlotMenuItem}
          <PMenu.Divider />
          <PMenu.Item itemKey="remove">
            <Icon.Close />
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
