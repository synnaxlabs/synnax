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
  ContextMenu as PContextMenu,
  type Flux,
  Icon,
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

export interface SnapshotContextMenuItemProps
  extends Omit<PContextMenu.ItemProps, "children"> {
  range?: Range | null;
}

export const SnapshotContextMenuItem = ({
  range,
  ...rest
}: SnapshotContextMenuItemProps): ReactElement | null =>
  range?.persisted === true ? (
    <PContextMenu.Item {...rest}>
      <Icon.Snapshot />
      Snapshot to {range.name}
    </PContextMenu.Item>
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

export interface SetAsActiveContextMenuItemProps
  extends Pick<PContextMenu.ItemProps, "onClick"> {}

export const SetAsActiveContextMenuItem = (
  props: SetAsActiveContextMenuItemProps,
): ReactElement => (
  <PContextMenu.Item {...props}>
    <Icon.Dynamic />
    Set as active range
  </PContextMenu.Item>
);

export interface ClearActiveContextMenuItemProps
  extends Pick<PContextMenu.ItemProps, "onClick"> {}

export const ClearActiveContextMenuItem = (
  props: ClearActiveContextMenuItemProps,
): ReactElement => (
  <PContextMenu.Item {...props}>
    <Icon.Dynamic />
    Clear active range
  </PContextMenu.Item>
);

export interface ViewDetailsContextMenuItemProps
  extends Pick<PContextMenu.ItemProps, "onClick"> {}

export const ViewDetailsContextMenuItem = (
  props: ViewDetailsContextMenuItemProps,
): ReactElement => (
  <PContextMenu.Item {...props}>
    <Icon.Details />
    View details
  </PContextMenu.Item>
);

const AddToNewPlotIcon = Icon.createComposite(Icon.LinePlot, {
  topRight: Icon.Add,
});

export interface AddToNewPlotContextMenuItemProps
  extends Pick<PContextMenu.ItemProps, "onClick"> {}

export const AddToNewPlotContextMenuItem = (
  props: AddToNewPlotContextMenuItemProps,
): ReactElement => (
  <PContextMenu.Item {...props}>
    <AddToNewPlotIcon />
    Add to new plot
  </PContextMenu.Item>
);

const AddToActivePlotIcon = Icon.createComposite(Icon.LinePlot, {
  topRight: Icon.Range,
});

export interface AddToActivePlotContextMenuItemProps
  extends Pick<PContextMenu.ItemProps, "onClick"> {}

export const AddToActivePlotContextMenuItem = (
  props: AddToActivePlotContextMenuItemProps,
): ReactElement => (
  <PContextMenu.Item {...props}>
    <AddToActivePlotIcon />
    Add to active plot
  </PContextMenu.Item>
);

export const CreateChildRangeIcon = Icon.createComposite(Icon.Range, {
  topRight: Icon.Add,
});

export interface CreateChildRangeContextMenuItemProps
  extends Pick<PContextMenu.ItemProps, "onClick"> {}

export const CreateChildRangeContextMenuItem = (
  props: CreateChildRangeContextMenuItemProps,
): ReactElement => (
  <PContextMenu.Item {...props}>
    <CreateChildRangeIcon />
    Create child range
  </PContextMenu.Item>
);

export const useViewDetails = (): ((key: string) => void) => {
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

export const ContextMenu = ({ keys: [key] }: PContextMenu.MenuProps) => {
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
  const copyLink = Cluster.useCopyLinkToClipboard();
  const handleLink = () => {
    if (!rangeExists) return;
    copyLink({ name: rng.name, ontologyID: ranger.ontologyID(rng.key) });
  };
  return (
    <>
      <PContextMenu.Item onClick={() => handleCreate()}>
        <Icon.Add />
        Create new
      </PContextMenu.Item>
      {rangeExists && (
        <>
          <PContextMenu.Divider />
          {rng.key !== activeRange?.key ? (
            <SetAsActiveContextMenuItem onClick={handleSetActive} />
          ) : (
            <ClearActiveContextMenuItem onClick={handleClearActive} />
          )}
          {rng.persisted && (
            <ViewDetailsContextMenuItem onClick={() => handleViewDetails(rng.key)} />
          )}
          <PContextMenu.Divider />
          <CMenu.RenameItem onClick={() => Text.edit(`text-${key}`)} />
          {rng.persisted && (
            <CreateChildRangeContextMenuItem onClick={handleAddChildRange} />
          )}
          <PContextMenu.Divider />
          {activeLayout?.type === LINE_PLOT_LAYOUT_TYPE && (
            <AddToActivePlotContextMenuItem onClick={() => addToActivePlot([key])} />
          )}
          <AddToNewPlotContextMenuItem onClick={() => addToNewPlot([key])} />
          <PContextMenu.Divider />
          <PContextMenu.Item onClick={() => handleRemove([rng.key])}>
            <Icon.Close />
            Remove from favorites
          </PContextMenu.Item>
          {rng.persisted ? (
            <>
              <CMenu.DeleteItem onClick={() => del(rng.key)} />
              <PContextMenu.Divider />
              <Link.CopyContextMenuItem onClick={handleLink} />
            </>
          ) : (
            client != null && (
              <>
                <PContextMenu.Divider />
                <PContextMenu.Item onClick={() => persist(rng.key)}>
                  <Icon.Save />
                  {`Save to ${client.props.name ?? "Synnax"}`}
                </PContextMenu.Item>
              </>
            )
          )}
        </>
      )}
      <PContextMenu.Divider />
      <CMenu.ReloadConsoleItem />
    </>
  );
};
