// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { lineplot, ranger, type Synnax as Client } from "@synnaxlabs/client";
import {
  Access,
  type Flux,
  Icon,
  Menu,
  Ranger,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { ContextMenu as CMenu } from "@/components";
import { Layout } from "@/layout";
import { LAYOUT_TYPE as LINE_PLOT_LAYOUT_TYPE } from "@/lineplot/layout";
import { Link } from "@/link";
import { useConfirmDelete } from "@/ontology/hooks";
import { createCreateLayout } from "@/range/Create";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";
import { select, useSelect, useSelectMultiple } from "@/range/selectors";
import { add, type Range, remove, setActive, type StoreState } from "@/range/slice";
import { fromClientRange } from "@/range/translate";
import { useAddToActivePlot } from "@/range/useAddToActivePlot";
import { useAddToNewPlot } from "@/range/useAddToNewPlot";

export interface SnapshotMenuItemProps {
  range?: Range | null;
  onClick?: () => void;
}

export const SnapshotMenuItem = ({
  range,
  onClick,
}: SnapshotMenuItemProps): ReactElement | null =>
  range?.persisted === true ? (
    <Menu.Item itemKey="rangeSnapshot" onClick={onClick}>
      <Icon.Snapshot />
      Snapshot to {range.name}
    </Menu.Item>
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

const AddToNewPlotIcon = Icon.createComposite(Icon.LinePlot, {
  topRight: Icon.Add,
});

const AddToActivePlotIcon = Icon.createComposite(Icon.LinePlot, {
  topRight: Icon.Range,
});

export const CreateChildRangeIcon = Icon.createComposite(Icon.Range, {
  topRight: Icon.Add,
});

const useViewDetails = (): ((key: string) => void) => {
  const addStatus = Status.useAdder();
  const placeLayout = Layout.usePlacer();
  const { retrieve } = Ranger.useRetrieveObservable({
    onChange: useCallback(
      ({ data, variant, status }) => {
        if (variant !== "success") {
          if (variant === "error") addStatus(status);
          return;
        }
        placeLayout({ ...OVERVIEW_LAYOUT, name: data.name, key: data.key });
      },
      [placeLayout],
    ),
  });
  return useCallback((key: string) => retrieve({ key }), [retrieve]);
};

const useDelete = () => {
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
      async ({ data }: Flux.BeforeUpdateParams<Ranger.DeleteParams>) => {
        const keys = array.toArray(data);
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

const usePersist = () => {
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

export const ContextMenu = ({ keys: [key] }: Menu.ContextMenuMenuProps) => {
  const dispatch = useDispatch();
  const client = Synnax.use();
  const ranges = useSelectMultiple();
  const id = ranger.ontologyID(key ?? "");
  const canEdit = Access.useUpdateGranted(id);
  const canDelete = Access.useDeleteGranted(id);
  const canUpdateLinePlot = Access.useUpdateGranted(lineplot.TYPE_ONTOLOGY_ID);
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

  return (
    <CMenu.Menu>
      {canEdit && (
        <Menu.Item itemKey="create" onClick={() => handleCreate()}>
          <Icon.Add />
          Create new
        </Menu.Item>
      )}
      {rangeExists && (
        <>
          {rng.key !== activeRange?.key ? (
            <Menu.Item itemKey="setAsActive" gap="small" onClick={handleSetActive}>
              <Icon.Dynamic />
              Set as active range
            </Menu.Item>
          ) : (
            <Menu.Item itemKey="clearActive" gap="small" onClick={handleClearActive}>
              <Icon.Dynamic />
              Clear active range
            </Menu.Item>
          )}
          {rng.persisted && (
            <Menu.Item itemKey="details" onClick={() => handleViewDetails(rng.key)}>
              <Icon.Details />
              View details
            </Menu.Item>
          )}
          {canEdit && (
            <>
              <Menu.Divider />
              <CMenu.RenameItem onClick={() => Text.edit(`text-${key}`)} />
              {rng.persisted && (
                <Menu.Item itemKey="addChildRange" onClick={handleAddChildRange}>
                  <CreateChildRangeIcon key="plot" />
                  Create child range
                </Menu.Item>
              )}
            </>
          )}
          <Menu.Divider />
          {activeLayout?.type === LINE_PLOT_LAYOUT_TYPE && canUpdateLinePlot && (
            <Menu.Item itemKey="addToActivePlot" onClick={() => addToActivePlot([key])}>
              <AddToActivePlotIcon key="plot" />
              Add to active plot
            </Menu.Item>
          )}
          {canUpdateLinePlot && (
            <Menu.Item itemKey="addToNewPlot" onClick={() => addToNewPlot([key])}>
              <AddToNewPlotIcon key="plot" />
              Add to new plot
            </Menu.Item>
          )}
          <Menu.Divider />
          <Menu.Item itemKey="remove" onClick={() => handleRemove([rng.key])}>
            <Icon.Close />
            Remove from favorites
          </Menu.Item>
          {rng.persisted ? (
            <>
              {canDelete && <CMenu.DeleteItem onClick={() => del(rng.key)} />}
              <Menu.Divider />
              <Link.CopyContextMenuItem
                onClick={() =>
                  handleLink({ name: rng.name, ontologyID: ranger.ontologyID(rng.key) })
                }
              />
            </>
          ) : (
            canEdit &&
            client != null && (
              <>
                <Menu.Divider />
                <Menu.Item itemKey="save" onClick={() => persist(rng.key)}>
                  <Icon.Save />
                  Save to Synnax
                </Menu.Item>
              </>
            )
          )}
        </>
      )}
      <Menu.Divider />
      <CMenu.ReloadConsoleItem />
    </CMenu.Menu>
  );
};
