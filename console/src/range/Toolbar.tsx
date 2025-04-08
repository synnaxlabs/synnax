// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Toolbar.css";

import { type Store } from "@reduxjs/toolkit";
import { type label, ranger, type Synnax as Client } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  componentRenderProp,
  Icon as PIcon,
  List as CoreList,
  Menu as PMenu,
  Ranger,
  Status,
  Synnax,
  Tag,
  Text,
  Tooltip,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { errors, toArray } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";
import { useDispatch, useStore } from "react-redux";

import { Cluster } from "@/cluster";
import { Menu, Toolbar } from "@/components";
import { CSS } from "@/css";
import { NULL_CLIENT_ERROR } from "@/errors";
import { Layout } from "@/layout";
import {
  create as createLinePlot,
  LAYOUT_TYPE as LINE_PLOT_LAYOUT_TYPE,
} from "@/lineplot/layout";
import { setRanges as setLinePlotRanges } from "@/lineplot/slice";
import { Link } from "@/link";
import { Modals } from "@/modals";
import { CREATE_LAYOUT, createCreateLayout } from "@/range/Create";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";
import { select, useSelect, useSelectMultiple } from "@/range/selectors";
import {
  add,
  type Range,
  remove,
  rename,
  setActive,
  type StaticRange,
  type StoreState,
} from "@/range/slice";
import { type RootState } from "@/store";

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

export const fromClientRange = (ranges: ranger.Range | ranger.Range[]): Range[] =>
  toArray(ranges).map((range) => ({
    variant: "static",
    key: range.key,
    name: range.name,
    timeRange: range.timeRange.numeric,
    persisted: true,
  }));

const fetchIfNotInState = async (
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

const useViewDetails = (): ((key: string) => void) => {
  const store = useStore<RootState>();
  const client = Synnax.use();
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

export const useAddToNewPlot = (): ((key: string) => void) => {
  const store = useStore<RootState>();
  const client = Synnax.use();
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

interface NoRangesProps {
  onLinkClick: (key?: string) => void;
}

const NoRanges = ({ onLinkClick }: NoRangesProps): ReactElement => {
  const handleLinkClick: React.MouseEventHandler<HTMLParagraphElement> = (e) => {
    e.stopPropagation();
    onLinkClick();
  };

  return (
    <Align.Space empty style={{ height: "100%", position: "relative" }}>
      <Align.Center y style={{ height: "100%" }} size="small">
        <Text.Text level="p">No ranges added.</Text.Text>
        <Text.Link level="p" onClick={handleLinkClick}>
          Add a range
        </Text.Link>
      </Align.Center>
    </Align.Space>
  );
};

const List = (): ReactElement => {
  const menuProps = PMenu.useContextMenu();
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const remover = Layout.useRemover();
  const dispatch = useDispatch();
  const ranges = useSelectMultiple();
  const activeRange = useSelect();

  const handleCreate = (key?: string): void => {
    placeLayout(createCreateLayout({ key }));
  };

  const handleRemove = (keys: string[]): void => {
    dispatch(remove({ keys }));
  };

  const handleSelect = (key: string): void => {
    dispatch(setActive(key));
  };

  const handleError = Status.useErrorHandler();

  const confirm = Modals.useConfirm();
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

  const ContextMenu = ({ keys: [key] }: PMenu.ContextMenuMenuProps) => {
    const rng = ranges.find((r) => r.key === key);
    const activeLayout = Layout.useSelectActiveMosaicLayout();
    const addToActivePlot = useAddToActivePlot();
    const addToNewPlot = useAddToNewPlot();
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

  return (
    <CoreList.List<string, StaticRange>
      data={ranges.filter((r) => r.variant === "static")}
      emptyContent={<NoRanges onLinkClick={handleCreate} />}
    >
      <PMenu.ContextMenu menu={(p) => <ContextMenu {...p} />} {...menuProps}>
        <CoreList.Selector
          value={activeRange?.key ?? null}
          onChange={handleSelect}
          allowMultiple={false}
          allowNone={true}
        >
          <CoreList.Core
            style={{ height: "100%", overflowX: "hidden" }}
            onContextMenu={menuProps.open}
            className={menuProps.className}
          >
            {componentRenderProp(ListItem)}
          </CoreList.Core>
        </CoreList.Selector>
      </PMenu.ContextMenu>
    </CoreList.List>
  );
};

interface ListItemProps extends CoreList.ItemProps<string, StaticRange> {}

const ListItem = (props: ListItemProps): ReactElement => {
  const { entry } = props;
  const client = Synnax.use();
  const dispatch = useDispatch();
  const [labels, setLabels] = useState<label.Label[]>([]);
  useAsyncEffect(async () => {
    if (client == null || labels.length > 0 || !entry.persisted) return;
    const labels_ = await (await client.ranges.retrieve(entry.key)).labels();
    setLabels(labels_);
  }, [entry.key, client]);
  const handleError = Status.useErrorHandler();
  const onRename = (name: string): void => {
    if (name.length === 0) return;
    dispatch(rename({ key: entry.key, name }));
    dispatch(Layout.rename({ key: entry.key, name }));
    if (!entry.persisted) return;
    client?.ranges
      .rename(entry.key, name)
      .catch((e) => handleError(e, "Failed to rename range"));
  };
  return (
    <CoreList.ItemFrame className={CSS.B("range-list-item")} {...props} size="small" y>
      {!entry.persisted && (
        <Tooltip.Dialog location="left">
          <Text.Text level="small">This range is local.</Text.Text>
          <Text.Text className="save-button" weight={700} level="small" shade={11}>
            L
          </Text.Text>
        </Tooltip.Dialog>
      )}
      <Text.MaybeEditable
        id={`text-${entry.key}`}
        level="p"
        value={entry.name}
        onChange={onRename}
        allowDoubleClick={false}
      />
      <Ranger.TimeRangeChip level="small" timeRange={entry.timeRange} />
      {labels.length > 0 && (
        <Align.Space
          x
          size="small"
          wrap
          style={{ overflowX: "auto", height: "fit-content" }}
        >
          {labels.map((l) => (
            <Tag.Tag key={l.key} size="small" color={l.color}>
              {l.name}
            </Tag.Tag>
          ))}
        </Align.Space>
      )}
    </CoreList.ItemFrame>
  );
};

const Content = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  return (
    <Align.Space empty style={{ height: "100%" }}>
      <Toolbar.Header>
        <Toolbar.Title icon={<Icon.Range />}>Ranges</Toolbar.Title>
        <Toolbar.Actions>
          {[{ children: <Icon.Add />, onClick: () => placeLayout(CREATE_LAYOUT) }]}
        </Toolbar.Actions>
      </Toolbar.Header>
      <List />
    </Align.Space>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "range",
  icon: <Icon.Range />,
  content: <Content />,
  tooltip: "Ranges",
  trigger: ["R"],
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
};
