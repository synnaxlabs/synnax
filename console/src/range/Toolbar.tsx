// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Toolbar.css";

import { type label, TimeRange } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  componentRenderProp,
  Header,
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
import { errors, id } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { Menu } from "@/components/menu";
import { Confirm } from "@/confirm";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { createEditLayout } from "@/range/EditLayout";
import { useSelect, useSelectMultiple } from "@/range/selectors";
import {
  add,
  type Range,
  remove,
  rename,
  setActive,
  type StaticRange,
} from "@/range/slice";

export const List = (): ReactElement => {
  const menuProps = PMenu.useContextMenu();
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const dispatch = useDispatch();
  const ranges = useSelectMultiple();
  const selectedRange = useSelect();

  const handleAddOrEdit = (key?: string): void => {
    const layout = createEditLayout(key == null ? "Range.Create" : "Range.Edit");
    placeLayout({ ...layout, key: key ?? layout.key });
  };

  const handleRemove = (keys: string[]): void => {
    dispatch(remove({ keys }));
  };

  const handleSelect = (key: string): void => {
    dispatch(setActive(key));
  };

  const addStatus = Status.useAggregator();

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
      return rng;
    },
    mutationFn: async (key: string) => await client?.ranges.delete(key),
    onError: (e, _, range) => {
      if (errors.CANCELED.matches(e)) return;
      addStatus({
        key: id.id(),
        variant: "error",
        message: "Failed to rename range",
        description: e.message,
      });
      dispatch(add({ ranges: [range as Range] }));
    },
  });

  const save = useMutation<void, Error, string, Range | undefined>({
    mutationFn: async (key: string) => {
      const range = ranges.find((r) => r.key === key);
      if (range == null || range.variant === "dynamic") return;
      await client?.ranges.create({
        key: range.key,
        timeRange: new TimeRange(range.timeRange.start, range.timeRange.end),
        name: range.name,
      });
      dispatch(add({ ranges: [{ ...range, persisted: true }] }));
    },
    onError: (e, _, range) => {
      addStatus({
        key: id.id(),
        variant: "error",
        message: "Failed to save range",
        description: e.message,
      });
      dispatch(add({ ranges: [range as Range] }));
    },
  });

  const handleLink = Link.useCopyToClipboard();

  const NoRanges = (): ReactElement => {
    const handleLinkClick: React.MouseEventHandler<HTMLParagraphElement> = (e) => {
      e.stopPropagation();
      handleAddOrEdit();
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

  const ContextMenu = ({
    keys: [key],
  }: PMenu.ContextMenuMenuProps): ReactElement | null => {
    const rng = ranges.find((r) => r.key === key);

    const handleSelect = {
      rename: (): void => Text.edit(`text-${key}`),
      create: () => handleAddOrEdit(),
      edit: () => handleAddOrEdit(rng?.key),
      remove: () => rng != null && handleRemove([rng.key]),
      delete: () => rng != null && del.mutate(rng.key),
      save: () => rng != null && save.mutate(rng.key),
      link: () =>
        rng != null &&
        handleLink({
          name: rng.name,
          resource: {
            key: rng.key,
            type: "range",
          },
        }),
    };

    return (
      <PMenu.Menu onChange={handleSelect} level="small" iconSpacing="small">
        <PMenu.Item startIcon={<Icon.Add />} itemKey="create">
          Create New
        </PMenu.Item>
        <PMenu.Divider />
        {rng != null && (
          <>
            <Menu.RenameItem />
            <PMenu.Item startIcon={<Icon.Edit />} itemKey="edit">
              Edit
            </PMenu.Item>
            <PMenu.Divider />
            <PMenu.Item startIcon={<Icon.Close />} itemKey="remove">
              Remove from List
            </PMenu.Item>
            {rng.persisted ? (
              <PMenu.Item startIcon={<Icon.Delete />} itemKey="delete">
                Delete
              </PMenu.Item>
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
            {rng.persisted && (
              <>
                <PMenu.Divider />
                <Link.CopyMenuItem />
              </>
            )}
            <PMenu.Divider />
          </>
        )}
        <Menu.HardReloadItem />
      </PMenu.Menu>
    );
  };

  return (
    <PMenu.ContextMenu menu={(p) => <ContextMenu {...p} />} {...menuProps}>
      <Core.List<string, StaticRange>
        data={ranges.filter((r) => r.variant === "static") as StaticRange[]}
        emptyContent={<NoRanges />}
      >
        <Core.Selector
          value={selectedRange?.key ?? null}
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
      <Ranger.TimeRangeChip timeRange={entry.timeRange} />
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
  const p = Layout.usePlacer();
  return (
    <Align.Space empty style={{ height: "100%" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Range />}>Ranges</ToolbarTitle>
        <Header.Actions>
          {[
            {
              children: <Icon.Add />,
              onClick: () => p(createEditLayout("Range.Create")),
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
