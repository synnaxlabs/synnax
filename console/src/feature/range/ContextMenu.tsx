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
import { useCallback } from "react";

import { Cluster } from "@/platform/cluster";
import { ContextMenu as CMenu } from "@/platform/context-menu";
import { Layout } from "@/platform/layout";
import { LinePlot } from "@/platform/lineplot";
import { Link } from "@/platform/link";
import { Ontology } from "@/platform/ontology";
import { Range } from "@/platform/range";
import { useAddToActivePlot } from "@/platform/range/useAddToActivePlot";
import { useAddToNewPlot } from "@/platform/range/useAddToNewPlot";
import { Session } from "@/session";


export const fetchIfNotInState = async (
  store: Store<Session.Range.StoreState>,
  client: Client,
  key: string,
): Promise<Session.Range.State> => {
  const existing = Session.Range.selectState(store.getState(), key);
  if (existing == null) {
    const ranges = Session.Range.fromClient(await client.ranges.retrieve(key));
    store.dispatch(Session.Range.add(ranges));
    return ranges[0];
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
        placeLayout({ ...Range.OVERVIEW_LAYOUT, name: data.name, key: data.key });
      },
      [placeLayout],
    ),
  });
  return useCallback((key: string) => retrieve({ key }), [retrieve]);
};

const useDelete = () => {
  const dispatch = Session.useDispatch();
  const remover = Layout.useRemover();
  const ranges = Session.Range.useSelectMultiple();
  const handleRemove = (keys: string[]): void => {
    dispatch(Session.Range.remove({ keys }));
  };
  const confirm = Ontology.useConfirmDelete({
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
  const dispatch = Session.useDispatch();
  const ranges = Session.Range.useSelectMultiple();
  const { update } = Ranger.useCreate();
  return useCallback(
    (key: string) => {
      const range = ranges.find((r) => r.key === key);
      if (range == null || range.variant === "dynamic") return;
      dispatch(Session.Range.add({ ...range, persisted: true }));
      update(range);
    },
    [dispatch, ranges],
  );
};

export const ContextMenu = ({ keys: [key] }: Menu.ContextMenuMenuProps) => {
  const dispatch = Session.useDispatch();
  const client = Synnax.use();
  const ranges = Session.Range.useSelectMultiple();
  const id = ranger.ontologyID(key ?? "");
  const hasCreatePermission = Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID);
  const hasUpdatePermission = Access.useUpdateGranted(id);
  const hasDeletePermission = Access.useDeleteGranted(id);
  const hasLinePlotUpdatePermission = Access.useUpdateGranted(
    lineplot.TYPE_ONTOLOGY_ID,
  );
  const hasLinePlotCreatePermission = Access.useCreateGranted(
    lineplot.TYPE_ONTOLOGY_ID,
  );

  const handleRemove = (keys: string[]): void => {
    dispatch(Session.Range.remove({ keys }));
  };

  const rng = ranges.find((r) => r.key === key);
  const activeLayout = Session.Layout.useSelectActiveMosaicLayout();
  const addToActivePlot = useAddToActivePlot();
  const addToNewPlot = useAddToNewPlot();
  const activeRange = Session.Range.useSelectState();
  const openCreate = Range.useCreateModal();
  const handleSetActive = () => {
    dispatch(Session.Range.select(key));
  };
  const handleClearActive = () => {
    dispatch(Session.Range.clearSelected());
  };
  const handleViewDetails = useViewDetails();
  const handleAddChildRange = () => {
    openCreate({ parent: key });
  };

  const rangeExists = rng != null;
  const del = useDelete();
  const persist = usePersist();
  const handleLink = Cluster.useCopyLinkToClipboard();

  return (
    <CMenu.Menu>
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
          {hasUpdatePermission && (
            <>
              <Menu.Divider />
              <CMenu.RenameItem onClick={() => Text.edit(`text-${key}`)} />
            </>
          )}
          {hasCreatePermission && rng.persisted && (
            <Menu.Item itemKey="addChildRange" onClick={handleAddChildRange}>
              <CreateChildRangeIcon key="plot" />
              Create child range
            </Menu.Item>
          )}
          <Menu.Divider />
          {activeLayout?.type === LinePlot.LAYOUT_TYPE &&
            hasLinePlotUpdatePermission && (
              <Menu.Item
                itemKey="addToActivePlot"
                onClick={() => addToActivePlot([key])}
              >
                <AddToActivePlotIcon key="plot" />
                Add to active plot
              </Menu.Item>
            )}
          {hasLinePlotCreatePermission && (
            <Menu.Item itemKey="addToNewPlot" onClick={() => addToNewPlot([key])}>
              <AddToNewPlotIcon key="plot" />
              Add to new plot
            </Menu.Item>
          )}
          <Menu.Divider />
          <Menu.Item itemKey="remove" onClick={() => handleRemove([rng.key])}>
            <Icon.Close />
            Unfavorite
          </Menu.Item>
          {rng.persisted ? (
            <>
              {hasDeletePermission && <CMenu.DeleteItem onClick={() => del(rng.key)} />}
              <Menu.Divider />
              <Link.CopyContextMenuItem
                onClick={() =>
                  handleLink({ name: rng.name, ontologyID: ranger.ontologyID(rng.key) })
                }
              />
            </>
          ) : (
            hasCreatePermission &&
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
