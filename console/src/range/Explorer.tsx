import { type label, ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  componentRenderProp,
  Divider,
  Dropdown,
  Haul,
  List,
  Menu,
  Ranger,
  Synnax,
  Tag,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

import { Layout } from "@/layout";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";

export const EXPLORER_LAYOUT_TYPE = "explorer";

export const EXPLORER_LAYOUT: Layout.BaseState = {
  type: EXPLORER_LAYOUT_TYPE,
  name: "Explorer",
  location: "mosaic",
  icon: "Explore",
};

interface ExplorerListItemProps extends List.ItemFrameProps<string, ranger.Payload> {}

const ExplorerListItem = (props: ExplorerListItemProps) => {
  const { entry } = props;
  const placeLayout = Layout.usePlacer();
  const client = Synnax.use();
  const [labels, setLabels] = useState<label.Label[]>([]);
  useAsyncEffect(async () => {
    const labels = await client?.labels.retrieveFor(ranger.ontologyID(entry.key));
    setLabels(labels ?? []);
  }, [entry.key]);
  return (
    <List.ItemFrame
      onClick={() =>
        placeLayout({ ...OVERVIEW_LAYOUT, name: entry.name, key: entry.key })
      }
      x
      size="tiny"
      justify="spaceBetween"
      align="center"
      style={{ padding: "1.5rem 3rem", width: "100%" }}
      draggable
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
      <Align.Space x>
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

export const Explorer: Layout.Renderer = () => {
  const client = Synnax.use();
  const drag = Haul.useDrag({ type: ranger.ONTOLOGY_TYPE, key: "cat" });
  const item = useCallback(
    ({ key, ...rest }: ExplorerListItemProps) => {
      const { entry } = rest;
      return (
        <ExplorerListItem
          key={key}
          onDragStart={(e) => {
            console.log("dragging");
            drag.startDrag(
              [{ key: entry.key, type: ranger.ONTOLOGY_TYPE, data: entry }],
              () => {},
            );
          }}
          onDragEnd={drag.onDragEnd}
          {...rest}
        />
      );
    },
    [drag.startDrag],
  );
  return (
    <List.List>
      <List.Search searcher={client?.ranges}>{() => <></>}</List.Search>
      <Filters />
      <List.Core>{item}</List.Core>
    </List.List>
  );
};

export const Filters = () => (
  <Align.Space
    x
    style={{ height: "6rem", padding: "0 2rem", borderBottom: "var(--pluto-border)" }}
    background={1}
    align="center"
  >
    <FilterDropdown />
    <Divider.Divider y />
  </Align.Space>
);

const FilterDropdown = () => {
  const dropdownProps = Dropdown.use();
  return (
    <Dropdown.Dialog
      {...dropdownProps}
      variant="floating"
      location={location.BOTTOM_LEFT}
    >
      <Button.Icon onClick={dropdownProps.toggle} shade={1}>
        <Icon.Filter />
      </Button.Icon>
      <Align.Space size="small" style={{ padding: "1rem" }}>
        <Menu.Menu>
          <Menu.Item itemKey="label" startIcon={<Icon.Time />}>
            Time
          </Menu.Item>
          <Menu.Item itemKey="label" startIcon={<Icon.Label />}>
            Labels
          </Menu.Item>
        </Menu.Menu>
      </Align.Space>
    </Dropdown.Dialog>
  );
};
