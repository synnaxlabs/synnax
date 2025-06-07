// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Explorer.css";

import { ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Haul,
  Icon as PIcon,
  Input,
  List,
  Menu as PMenu,
  Ranger,
  type RenderProp,
  Synnax,
  Tag,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { useCallback, useRef } from "react";
import { useDispatch } from "react-redux";

import { Menu } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import {
  deleteMenuItem,
  useDelete,
  useLabels,
  useParent,
  useViewDetails,
  viewDetailsMenuItem,
} from "@/range/ContextMenu";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";
import { useSelectKeys } from "@/range/selectors";
import { add, remove } from "@/range/slice";
import { useRename } from "@/range/Toolbar";

export const EXPLORER_LAYOUT_TYPE = "explorer";

export const EXPLORER_LAYOUT: Layout.BaseState = {
  key: EXPLORER_LAYOUT_TYPE,
  type: EXPLORER_LAYOUT_TYPE,
  name: "Explorer",
  location: "mosaic",
  icon: "Explore",
};

interface ExplorerListItemProps
  extends Omit<List.ItemFrameProps<string, ranger.Payload>, "onDragEnd">,
    Pick<Haul.UseDragReturn, "startDrag" | "onDragEnd"> {}

const ExplorerListItem = ({
  startDrag,
  onDragEnd,
  selected,
  ...props
}: ExplorerListItemProps) => {
  const { entry } = props;
  const placeLayout = Layout.usePlacer();
  const labels = useLabels(entry.key);
  const parent = useParent(entry.key);
  const dragGhost = useRef<HTMLElement | null>(null);
  const elRef = useRef<HTMLDivElement>(null);
  const onRename = useRename(entry.key);
  const dispatch = useDispatch();
  const handleStar = () => {
    if (!selected)
      dispatch(
        add({
          ranges: [
            {
              key: entry.key,
              name: entry.name,
              variant: "static",
              persisted: true,
              timeRange: entry.timeRange.numeric,
            },
          ],
        }),
      );
    else dispatch(remove({ keys: [entry.key] }));
  };

  return (
    <List.ItemFrame
      ref={elRef}
      onClick={() =>
        placeLayout({ ...OVERVIEW_LAYOUT, name: entry.name, key: entry.key })
      }
      x
      size="tiny"
      justify="spaceBetween"
      align="center"
      className={CSS(CSS.B("range-explorer-item"))}
      style={{ padding: "0rem 3rem 0 2rem", width: "100%", height: "7rem" }}
      draggable
      selected={false}
      onDragStart={(e) => {
        const ghost = elRef.current?.cloneNode(true) as HTMLElement;
        ghost.classList.add("console--dragging");
        document.body.appendChild(ghost);
        dragGhost.current = ghost;
        e.dataTransfer.setDragImage(ghost, 50, 50);
        startDrag([
          {
            key: entry.key,
            type: ranger.ONTOLOGY_TYPE,
            data: {
              ...(entry as ranger.Range).payload,
              timeRange: entry.timeRange.numeric,
            },
          },
        ]);
      }}
      onDragEnd={(e) => {
        if (dragGhost.current) {
          document.body.removeChild(dragGhost.current);
          dragGhost.current = null;
        }
        onDragEnd(e);
      }}
      {...props}
    >
      <Align.Space x align="center">
        <Text.WithIcon
          startIcon={<Icon.Range style={{ color: "var(--pluto-gray-l9)" }} />}
          level="p"
          weight={450}
          shade={11}
          size="small"
          grow
          shrink={0}
        >
          <Text.Editable
            id={`explorer-${entry.key}`}
            level="p"
            value={entry.name}
            onChange={onRename}
          />
          {parent != null && (
            <Text.WithIcon
              level="p"
              shade={8}
              weight={450}
              startIcon={<Icon.Caret.Right />}
              style={{ marginTop: "1px" }}
              size="tiny"
            >
              {parent.name}
            </Text.WithIcon>
          )}
        </Text.WithIcon>
      </Align.Space>
      <Align.Space x className={CSS.B("range-explorer-item-content")} align="center">
        <Align.Stack x size="small">
          {labels.map((l) => (
            <Tag.Tag key={l.key} color={l.color} size="small">
              {l.name}
            </Tag.Tag>
          ))}
        </Align.Stack>
        <Ranger.TimeRangeChip level="p" timeRange={entry.timeRange} showSpan />
        <PIcon.Icon
          className={CSS(
            CSS.B("range-explorer-item-star"),
            selected && CSS.M("selected"),
          )}
          onClick={(e) => {
            e.stopPropagation();
            handleStar();
          }}
        >
          {selected ? <Icon.StarFilled /> : <Icon.StarOutlined />}
        </PIcon.Icon>
      </Align.Space>
    </List.ItemFrame>
  );
};

const ChangeLoader = () => {
  const { setSourceData } = List.useDataUtils<string>();
  const client = Synnax.use();
  useAsyncEffect(async () => {
    const obs = await client?.ranges.openTracker();
    obs?.onChange((changes) => {
      setSourceData((prev) => {
        const deletes = new Set(
          changes.filter((c) => c.variant === "delete").map((c) => c.key),
        );
        const next = prev.filter((r) => !deletes.has(r.key));
        const sets = changes.filter((c) => c.variant === "set");
        const setKeys = new Set(sets.map((c) => c.key));
        return [
          ...next.filter((r) => !setKeys.has(r.key)),
          ...sets.map((c) => c.value),
        ];
      });
    });
    return async () => {
      await obs?.close();
    };
  }, [client]);
  return null;
};

const ExplorerContextMenu = ({ keys: [key] }: PMenu.ContextMenuMenuProps) => {
  const details = useViewDetails();
  const del = useDelete();
  const handleSelect: PMenu.MenuProps["onChange"] = {
    details: () => details(key),
    delete: () => del.mutate(key),
    rename: () => Text.edit(`explorer-${key}`),
  };
  return (
    <PMenu.Menu level="small" onChange={handleSelect}>
      {viewDetailsMenuItem}
      <PMenu.Divider />
      {deleteMenuItem}
      <Menu.RenameItem />
      <PMenu.Divider />
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

export const Explorer: Layout.Renderer = () => {
  const client = Synnax.use();
  const drag = Haul.useDrag({ type: ranger.ONTOLOGY_TYPE, key: "cat" });
  const selected = useSelectKeys();
  const item: RenderProp<List.ItemProps<string, ranger.Payload> & { key: string }> =
    useCallback(
      ({ key, ...rest }) => (
        <ExplorerListItem
          key={key}
          {...drag}
          {...rest}
          selected={selected.includes(key)}
        />
      ),
      [drag.startDrag, selected],
    );
  const pMenuProps = PMenu.useContextMenu();
  const details = useViewDetails();

  return (
    <List.List>
      <PMenu.ContextMenu {...pMenuProps} menu={(p) => <ExplorerContextMenu {...p} />}>
        <ChangeLoader />
        <Align.Space x className={CSS.B("range-explorer-header")}>
          <List.Search searcher={client?.ranges}>
            {(p) => <Input.Text {...p} placeholder="Search Ranges" />}
          </List.Search>
        </Align.Space>
        <List.Selector
          allowMultiple={false}
          value={null}
          onChange={(key: string) => details(key)}
        >
          <List.Hover>
            <List.Core.Virtual
              onContextMenu={pMenuProps.open}
              className={pMenuProps.className}
              itemHeight={6 * 7}
              style={{ height: "calc(100% - 9rem)" }}
            >
              {item}
            </List.Core.Virtual>
          </List.Hover>
        </List.Selector>
      </PMenu.ContextMenu>
    </List.List>
  );
};
