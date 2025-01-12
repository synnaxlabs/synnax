// Copyright 2024 Synnax Labs, Inc.
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
  Header,
  Icon as PIcon,
  List as Core,
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

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Menu } from "@/components/menu";
import { Confirm } from "@/confirm";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { create as createLinePlot } from "@/lineplot/LinePlot";
import { setRanges as setLinePlotRanges } from "@/lineplot/slice";
import { Link } from "@/link";
import { createLayout } from "@/range/CreateLayout";
import { overviewLayout } from "@/range/external";
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
        <Icon.Visualize key="plot" />
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
        <Icon.Visualize key="plot" />
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
    Set as Active
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
  const handleException = Status.useHandleException();
  return useMutation<void, Error, string>({
    mutationKey: ["add-to-active-plot", client?.key],
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
    onError: (e) => handleException(e, "Failed to add range to plot"),
  }).mutate;
};

const useViewDetails = (): ((key: string) => void) => {
  const store = useStore<RootState>();
  const client = Synnax.use();
  const handleException = Status.useHandleException();
  const place = Layout.usePlacer();
  return useMutation<void, Error, string>({
    mutationFn: async (key: string) => {
      if (client == null) return;
      const rng = await fetchIfNotInState(store, client, key);
      place({ ...overviewLayout, name: rng.name, key: rng.key });
    },
    onError: (e) => handleException(e, "Failed to view details"),
  }).mutate;
};

export const useAddToNewPlot = (): ((key: string) => void) => {
  const store = useStore<RootState>();
  const client = Synnax.use();
  const place = Layout.usePlacer();
  const handleException = Status.useHandleException();
  return useMutation<void, Error, string>({
    mutationFn: async (key: string) => {
      if (client == null) return;
      const res = await fetchIfNotInState(store, client, key);
      place(
        createLinePlot({
          name: `Plot for ${res.name}`,
          ranges: { x1: [key], x2: [] },
        }),
      );
    },
    onError: (e) => handleException(e, "Failed to add range to new plot"),
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
      <Align.Center direction="y" style={{ height: "100%" }} size="small">
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
  const place = Layout.usePlacer();
  const remover = Layout.useRemover();
  const dispatch = useDispatch();
  const ranges = useSelectMultiple();
  const activeRange = useSelect();

  const handleCreate = (key?: string): void => {
    place(createLayout({ initial: { key } }));
  };

  const handleRemove = (keys: string[]): void => {
    dispatch(remove({ keys }));
  };

  const handleSelect = (key: string): void => {
    dispatch(setActive(key));
  };

  const handleException = Status.useHandleException();

  const confirm = Confirm.useModal();
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
      handleException(e, "Failed to delete range");
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
    onError: (e) => handleException(e, "Failed to save range"),
  });

  const handleLink = Link.useCopyToClipboard();

  const ContextMenu = ({
    keys: [key],
  }: PMenu.ContextMenuMenuProps): ReactElement | null => {
    const rng = ranges.find((r) => r.key === key);
    const activeLayout = Layout.useSelectActiveMosaicLayout();
    const addToActivePlot = useAddToActivePlot();
    const addToNewPlot = useAddToNewPlot();
    const place = Layout.usePlacer();
    const handleSetActive = () => {
      dispatch(setActive(key));
    };
    const handleViewDetails = useViewDetails();
    const handleAddChildRange = () => {
      place(createLayout({ initial: { parent: key } }));
    };

    const rangeExists = rng != null;

    const handleSelect: Record<string, () => void> = {
      rename: () => Text.edit(`text-${key}`),
      create: () => handleCreate(),
      remove: () => rangeExists && handleRemove([rng.key]),
      delete: () => rangeExists && del.mutate(rng.key),
      details: () => rangeExists && handleViewDetails(rng.key),
      save: () => rangeExists && save.mutate(rng.key),
      link: () =>
        rangeExists &&
        handleLink({
          name: rng.name,
          ontologyID: {
            key: rng.key,
            type: ranger.ONTOLOGY_TYPE,
          },
        }),
      addToActivePlot: () => addToActivePlot(key),
      addToNewPlot: () => addToNewPlot(key),
      addChildRange: handleAddChildRange,
      setAsActive: handleSetActive,
    };

    return (
      <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
        <PMenu.Item startIcon={<Icon.Add />} itemKey="create">
          Create New
        </PMenu.Item>
        {rangeExists && (
          <>
            <PMenu.Divider />
            {rng.key !== activeRange?.key && setAsActiveMenuItem}
            {rng.persisted && viewDetailsMenuItem}
            {(rng.key !== activeRange?.key || rng.persisted) && <PMenu.Divider />}
            <Menu.RenameItem />
            {rng.persisted && addChildRangeMenuItem}
            <PMenu.Divider />
            {activeLayout?.type === "lineplot" && addToActivePlotMenuItem}
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
    <PMenu.ContextMenu menu={(p) => <ContextMenu {...p} />} {...menuProps}>
      <Core.List<string, StaticRange>
        data={ranges.filter((r) => r.variant === "static")}
        emptyContent={<NoRanges onLinkClick={handleCreate} />}
      >
        <Core.Selector
          value={activeRange?.key ?? null}
          onChange={handleSelect}
          allowMultiple={false}
          allowNone={true}
        >
          <Core.Core style={{ height: "100%", overflowX: "hidden" }}>
            {componentRenderProp(ListItem)}
          </Core.Core>
        </Core.Selector>
      </Core.List>
    </PMenu.ContextMenu>
  );
};

interface ListItemProps extends Core.ItemProps<string, StaticRange> {}

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
  const onRename = (name: string): void => {
    if (name.length === 0) return;
    dispatch(rename({ key: entry.key, name }));
    dispatch(Layout.rename({ key: entry.key, name }));
    if (!entry.persisted) return;
    void (async () => {
      await client?.ranges.rename(entry.key, name);
    })();
  };
  return (
    <Core.ItemFrame
      className={CSS.B("range-list-item")}
      direction="y"
      rightAligned
      {...props}
      size="small"
    >
      {!entry.persisted && (
        <Tooltip.Dialog location={"left"}>
          <Text.Text level="small">This range is local.</Text.Text>
          <Text.Text className="save-button" weight={700} level="small" shade={7}>
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
          direction="x"
          size="small"
          wrap
          style={{
            overflowX: "auto",
            height: "fit-content",
          }}
        >
          {labels.map((l) => (
            <Tag.Tag key={l.key} size="small" color={l.color}>
              {l.name}
            </Tag.Tag>
          ))}
        </Align.Space>
      )}
    </Core.ItemFrame>
  );
};

const Content = (): ReactElement => {
  const place = Layout.usePlacer();
  return (
    <Align.Space empty style={{ height: "100%" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Range />}>Ranges</ToolbarTitle>
        <Header.Actions>
          {[
            {
              children: <Icon.Add />,
              onClick: () => place(createLayout({})),
            },
          ]}
        </Header.Actions>
      </ToolbarHeader>
      <List />
    </Align.Space>
  );
};

export const Toolbar: Layout.NavDrawerItem = {
  key: "range",
  icon: <Icon.Range />,
  content: <Content />,
  tooltip: "Ranges",
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
};
