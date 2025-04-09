import "@/range/Explorer.css";

import { type label, ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Haul,
  List,
  Menu as PMenu,
  Ranger,
  type RenderProp,
  Synnax,
  Tag,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { useCallback, useRef, useState } from "react";

import { Menu } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import {
  deleteMenuItem,
  useDelete,
  useViewDetails,
  viewDetailsMenuItem,
} from "@/range/ContextMenu";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";

export const EXPLORER_LAYOUT_TYPE = "explorer";

export const EXPLORER_LAYOUT: Layout.BaseState = {
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
  ...props
}: ExplorerListItemProps) => {
  const { entry } = props;
  const placeLayout = Layout.usePlacer();
  const client = Synnax.use();
  const [labels, setLabels] = useState<label.Label[]>([]);
  useAsyncEffect(async () => {
    if (client == null) return;
    const labels = await client.labels.retrieveFor(ranger.ontologyID(entry.key));
    setLabels(labels ?? []);
    const labelObs = await client.labels.trackLabelsOf(ranger.ontologyID(entry.key));
    labelObs?.onChange((changes) => {
      setLabels(changes);
    });
    return async () => {
      await labelObs?.close();
    };
  }, [entry.key]);
  const dragGhost = useRef<HTMLElement | null>(null);
  const elRef = useRef<HTMLDivElement>(null);

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
      style={{ padding: "1.5rem 3rem", width: "100%" }}
      draggable
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
      <Text.WithIcon
        startIcon={<Icon.Range style={{ color: "var(--pluto-gray-l11)" }} />}
        level="p"
        weight={450}
        shade={11}
        size="small"
        grow
        shrink={0}
      >
        {entry.name}
      </Text.WithIcon>
      <Align.Space x className={CSS.B("range-explorer-item-content")}>
        <Align.Stack x size="small">
          {labels.map((l) => (
            <Tag.Tag key={l.key} color={l.color} size="small">
              {l.name}
            </Tag.Tag>
          ))}
        </Align.Stack>
        <Ranger.TimeRangeChip level="p" timeRange={entry.timeRange} showSpan />
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
  }, [client]);
  return null;
};

const ExplorerContextMenu = ({ keys }: PMenu.ContextMenuMenuProps) => {
  const details = useViewDetails();
  const del = useDelete();
  const handleSelect: PMenu.MenuProps["onChange"] = {
    details: () => details(keys[0]),
    delete: () => del.mutate(keys[0]),
  };
  return (
    <PMenu.Menu level="small" onChange={handleSelect}>
      {viewDetailsMenuItem}
      {deleteMenuItem}
      <PMenu.Divider />
      <Menu.RenameItem />
    </PMenu.Menu>
  );
};
export const Explorer: Layout.Renderer = () => {
  const client = Synnax.use();
  const drag = Haul.useDrag({ type: ranger.ONTOLOGY_TYPE, key: "cat" });
  const item: RenderProp<List.ItemProps<string, ranger.Payload> & { key: string }> =
    useCallback(
      ({ key, ...rest }) => <ExplorerListItem key={key} {...drag} {...rest} />,
      [drag.startDrag],
    );
  const pMenuProps = PMenu.useContextMenu();

  return (
    <List.List>
      <PMenu.ContextMenu {...pMenuProps} menu={(p) => <ExplorerContextMenu {...p} />}>
        <ChangeLoader />
        <List.Search searcher={client?.ranges} />
        <List.Core onContextMenu={pMenuProps.open} className={pMenuProps.className}>
          {item}
        </List.Core>
      </PMenu.ContextMenu>
    </List.List>
  );
};
