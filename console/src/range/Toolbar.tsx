// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Toolbar.css";

import { type label, ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  componentRenderProp,
  Haul,
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
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";

import { Toolbar } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { ContextMenu } from "@/range/ContextMenu";
import { CREATE_LAYOUT, createCreateLayout } from "@/range/Create";
import { EXPLORER_LAYOUT } from "@/range/Explorer";
import { useSelect, useSelectMultiple } from "@/range/selectors";
import { add, rename, setActive, type StaticRange } from "@/range/slice";

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
  const dispatch = useDispatch();
  const activeRange = useSelect();
  const ranges = useSelectMultiple();

  const handleSelect = (key: string): void => {
    dispatch(setActive(key));
  };

  const placeLayout = Layout.usePlacer();
  const handleCreate = (key?: string): void => {
    placeLayout(createCreateLayout({ key }));
  };

  const drop = Haul.useDrop({
    type: "range-toolbar",
    canDrop: Haul.canDropOfType(ranger.ONTOLOGY_TYPE),
    onDrop: ({ items }) => {
      const ranges = items.map((e) => ({
        key: e.key,
        name: e.data?.name,
        variant: "static",
        persisted: true,
        timeRange: e.data?.timeRange,
      }));
      dispatch(add({ ranges }));
      return items;
    },
  });

  const menuProps = PMenu.useContextMenu();

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
            {...drop}
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
            <Tag.Tag key={l.key} size="tiny" color={l.color}>
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
      <Toolbar.Header align="center" style={{ paddingRight: "0.5rem" }}>
        <Toolbar.Title icon={<Icon.Range />}>Ranges</Toolbar.Title>
        <Align.Pack>
          <Button.Button
            variant="filled"
            size="small"
            onClick={() => placeLayout(EXPLORER_LAYOUT)}
            startIcon={<Icon.Explore />}
          >
            Explorer
          </Button.Button>
          <Button.Icon
            variant="outlined"
            size="small"
            onClick={() => placeLayout(CREATE_LAYOUT)}
          >
            <Icon.Add />
          </Button.Icon>
        </Align.Pack>
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
