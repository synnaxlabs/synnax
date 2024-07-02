// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/accordionEntry.css";

import { type label, TimeRange } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  componentRenderProp,
  Ranger,
  Status,
  Synnax,
  Tag,
  Tooltip,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { List as Core } from "@synnaxlabs/pluto/list";
import { Menu as PMenu } from "@synnaxlabs/pluto/menu";
import { Text } from "@synnaxlabs/pluto/text";
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";

import { Menu } from "@/components/menu";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { createEditLayout } from "@/range/EditLayout";
import type { StaticRange } from "@/range/range";
import { useSelect, useSelectMultiple } from "@/range/selectors";
import { add, remove, setActive } from "@/range/slice";

export const List = (): ReactElement => {
  const menuProps = PMenu.useContextMenu();
  const client = Synnax.use();
  const newLayout = Layout.usePlacer();
  const dispatch = useDispatch();
  const ranges = useSelectMultiple();
  const selectedRange = useSelect();
  const addStatus = Status.useAggregator();

  const handleAddOrEdit = (key?: string): void => {
    const layout = createEditLayout(key == null ? "Create Range" : "Edit Range");
    newLayout({
      ...layout,
      key: key ?? layout.key,
    });
  };

  const handleRemove = (keys: string[]): void => {
    dispatch(remove({ keys }));
  };

  const handleSelect = (key: string): void => {
    dispatch(setActive(key));
  };

  const handleDelete = (key: string): undefined => {
    void (async () => {
      await client?.ranges.delete(key);
      handleRemove([key]);
    })();
  };

  const handleSave = (key: string): undefined => {
    void (async () => {
      const range = ranges.find((r) => r.key === key);
      if (range == null || range.variant === "dynamic") return;
      await client?.ranges.create({
        key: range.key,
        timeRange: new TimeRange(range.timeRange.start, range.timeRange.end),
        name: range.name,
      });
      dispatch(add({ ranges: [{ ...range, persisted: true }] }));
    })();
  };

  const handleSetActive = (key: string): void => {
    void (async () => {
      await client?.ranges.setActive(key);
    })();
  };

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
    const handleClick = (itemKey: string): void => {
      switch (itemKey) {
        case "create":
          return handleAddOrEdit();
        case "edit":
          if (rng == null) return;
          return handleAddOrEdit(rng.key);
        case "remove":
          if (rng == null) return;
          return handleRemove([rng.key]);
        case "delete":
          if (rng == null) return;
          return handleDelete(rng.key);
        case "save":
          if (rng == null) return;
          return handleSave(rng.key);
        case "setActive":
          if (rng == null) return;
          return handleSetActive(rng.key);
        case "link": {
          if (rng == null) return;
          if (client == null) return;
          return Link.CopyLinkToClipboard({
            clusterKey: client.key,
            resource: {
              type: "range",
              key: rng.key,
            },
            name: rng.name,
            addStatus,
          });
        }
      }
    };
    return (
      <PMenu.Menu onChange={handleClick}>
        {rng != null && (
          <>
            <PMenu.Item startIcon={<Icon.Edit />} size="small" itemKey="edit">
              Edit
            </PMenu.Item>
            <PMenu.Item startIcon={<Icon.Close />} size="small" itemKey="remove">
              Remove from List
            </PMenu.Item>
            {rng.persisted ? (
              <PMenu.Item startIcon={<Icon.Delete />} size="small" itemKey="delete">
                Delete
              </PMenu.Item>
            ) : (
              client != null && (
                <PMenu.Item startIcon={<Icon.Save />} size="small" itemKey="save">
                  Save to Synnax
                </PMenu.Item>
              )
            )}
          </>
        )}
        <PMenu.Item startIcon={<Icon.Add />} size="small" itemKey="create">
          Create New
        </PMenu.Item>
        <PMenu.Item startIcon={<Icon.Play />} size="small" itemKey="setActive">
          Set as Active Range
        </PMenu.Item>
        {rng?.persisted && <Link.CopyMenuItem />}
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
  const [labels, setLabels] = useState<label.Label[]>([]);
  useAsyncEffect(async () => {
    if (client == null || labels.length > 0 || !entry.persisted) return;
    const labels_ = await (await client.ranges.retrieve(entry.key)).labels();
    setLabels(labels_);
  }, [entry.key, client]);
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

      <Text.WithIcon level="p" weight={500}>
        {entry.name}
      </Text.WithIcon>
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
