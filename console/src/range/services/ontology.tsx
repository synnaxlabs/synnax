// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import {
  type Flux,
  type Haul,
  Icon,
  List,
  Menu as PMenu,
  Ranger,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { type CrudeTimeRange, strings } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { useConfirmDelete } from "@/ontology/hooks";
import { type TreeContextMenuProps } from "@/ontology/service";
import {
  addChildRangeMenuItem,
  addToActivePlotMenuItem,
  addToNewPlotMenuItem,
  clearActiveMenuItem,
  deleteMenuItem,
  setAsActiveMenuItem,
  viewDetailsMenuItem,
} from "@/range/ContextMenu";
import { createCreateLayout } from "@/range/Create";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";
import { useSelect } from "@/range/selectors";
import { add, remove, rename, setActive } from "@/range/slice";
import { fromClientRange } from "@/range/translate";
import { useAddToActivePlot } from "@/range/useAddToActivePlot";
import { useAddToNewPlot } from "@/range/useAddToNewPlot";

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

const useActivate = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const addStatus = Status.useAdder();
  const dispatch = useDispatch();
  const { retrieve } = Ranger.useRetrieveObservable({
    onChange: ({ data, variant, status }) => {
      if (variant !== "success") {
        if (variant === "error") addStatus(status);
        return;
      }
      dispatch(add({ ranges: fromClientRange(data) }));
      dispatch(setActive(data.key));
    },
  });
  return useCallback(
    (props: Ontology.TreeContextMenuProps) => {
      const { key } = props.selection.ids[0];
      retrieve({ key });
    },
    [retrieve],
  );
};

const useViewDetails = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const placeLayout = Layout.usePlacer();
  return ({ selection: { ids }, state: { getResource } }) =>
    placeLayout({
      ...OVERVIEW_LAYOUT,
      name: getResource(ids[0]).name,
      key: ids[0].key,
    });
};

const useDelete = ({
  selection: { ids },
  state: { getResource },
  store,
  removeLayout,
}: TreeContextMenuProps): (() => void) => {
  const keys = useMemo(() => ids.map(({ key }) => key), [ids]);
  const confirm = useConfirmDelete({
    type: "Range",
    description: "Deleting this range will also delete all child ranges.",
  });
  const { update } = Ranger.useDelete({
    beforeUpdate: useCallback(async () => {
      const resources = getResource(ids);
      if (!(await confirm(resources))) return false;
      store.dispatch(remove({ keys }));
      removeLayout(...ontology.idToString(ids));
      return true;
    }, [ids, getResource, store, removeLayout, keys]),
  });
  return useCallback(() => update(keys), [keys]);
};

const useRename = ({
  selection: {
    ids: [firstID],
  },
  state: { getResource },
}: Ontology.TreeContextMenuProps) => {
  const dispatch = useDispatch();
  const { update } = Ranger.useRename({
    beforeUpdate: useCallback(
      async ({ data, rollbacks }: Flux.BeforeUpdateParams<Ranger.RenameParams>) => {
        const { name: oldName } = data;
        const [name, renamed] = await Text.asyncEdit(ontology.idToString(firstID));
        if (!renamed) return false;
        dispatch(Layout.rename({ key: firstID.key, name }));
        dispatch(rename({ key: firstID.key, name }));
        rollbacks.add(() => {
          dispatch(Layout.rename({ key: firstID.key, name: oldName }));
          dispatch(rename({ key: firstID.key, name: oldName }));
        });
        return { ...data, name };
      },
      [firstID],
    ),
  });
  return useCallback(
    () => update({ key: firstID.key, name: getResource(firstID).name }),
    [firstID],
  );
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    store,
    state: { getResource, shape },
    placeLayout,
  } = props;
  const activeRange = useSelect();
  const layout = Layout.useSelectActiveMosaicLayout();
  const keys = ids.map((id) => id.key);
  const handleDelete = useDelete(props);
  const rename = useRename(props);
  const addToActivePlot = useAddToActivePlot();
  const addToNewPlot = useAddToNewPlot();
  const activate = useActivate();
  const dispatch = useDispatch();
  const clearActiveRange = () => {
    dispatch(setActive(null));
  };
  const firstID = ids[0];
  const firstResource = getResource(firstID);
  const groupFromSelection = Group.useCreateFromSelection();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleAddChildRange = () => {
    placeLayout(createCreateLayout({ parent: firstID.key }));
  };
  const viewDetails = useViewDetails();
  const handleSelect = {
    delete: handleDelete,
    rename,
    setAsActive: () => activate(props),
    addToActivePlot: () => addToActivePlot(keys),
    addToNewPlot: () => addToNewPlot(keys),
    group: () => groupFromSelection(props),
    details: () => viewDetails(props),
    link: () => handleLink({ name: firstResource.name, ontologyID: firstID }),
    addChildRange: handleAddChildRange,
    clearActive: clearActiveRange,
  };
  const isSingle = ids.length === 1;
  let showAddToActivePlot = false;
  if (layout?.type === LinePlot.LAYOUT_TYPE) {
    const activeRanges = LinePlot.selectRanges(store.getState(), layout.key).x1.map(
      (r) => r.key,
    );
    showAddToActivePlot = ids.some((r) => !activeRanges.includes(r.key));
  }

  return (
    <PMenu.Menu onChange={handleSelect} level="small" gap="small">
      {isSingle && (
        <>
          {firstID.key !== activeRange?.key ? setAsActiveMenuItem : clearActiveMenuItem}
          {viewDetailsMenuItem}
          <PMenu.Divider />
          <Menu.RenameItem />
          {addChildRangeMenuItem}
          <PMenu.Divider />
        </>
      )}
      <Group.MenuItem ids={ids} shape={shape} rootID={rootID} />
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
  TreeContextMenu,
  PaletteListItem,
};
