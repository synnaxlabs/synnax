import { type ranger } from "@synnaxlabs/client";
import {
  Align,
  Component,
  Icon,
  Input,
  List,
  Menu as PMenu,
  Ranger,
  Select,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { compare } from "@synnaxlabs/x";
import { useState } from "react";
import { useDispatch } from "react-redux";

import { Menu } from "@/components";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";
import {
  addChildRangeMenuItem,
  deleteMenuItem,
  fromClientRange,
  useViewDetails,
} from "@/range/ContextMenu";
import { createCreateLayout } from "@/range/Create";
import { useSelectMultiple } from "@/range/selectors";
import { add, remove } from "@/range/slice";

export interface ListItemProps extends List.ItemProps<ranger.Key> {}

export const ListItem = (props: ListItemProps) => {
  const selected = useSelectMultiple();
  const dispatch = useDispatch();
  const { getItem } = List.useUtilContext<ranger.Key, ranger.Range>();
  const handleStar = (starred: boolean) => {
    const item = getItem?.(props.itemKey);
    if (item == null) return;
    if (starred) dispatch(add({ ranges: fromClientRange(item) }));
    else dispatch(remove({ keys: [item.key] }));
  };
  const viewDetails = useViewDetails();
  const client = Synnax.use();
  const handleStageChange = (stage: ranger.Stage) => {
    const item = getItem?.(props.itemKey);
    if (item == null) return;
    client?.ranges.create({
      ...item.payload,
      stage,
    });
  };
  return (
    <Ranger.ListItem
      {...props}
      starred={selected.map((s) => s.key).includes(props.itemKey)}
      onStar={handleStar}
      showAgo={false}
      onClick={() => viewDetails(props.itemKey)}
      showSpan
      onStageChange={handleStageChange}
    />
  );
};

const listItem = Component.renderProp(ListItem);

export const EXPLORER_LAYOUT_TYPE = "range_explorer";

interface ContextMenuProps extends PMenu.ContextMenuMenuProps {
  ranges: ranger.Range[];
}

const ContextMenu = ({ ranges }: ContextMenuProps) => {
  const isEmpty = ranges.length === 0;
  const isSingle = ranges.length === 1;
  const placeLayout = Layout.usePlacer();
  const rename = Modals.useRename();
  const confirm = useConfirmDelete({
    type: "Range",
    description: "Deleting this range will also delete all child ranges.",
  });
  const { update } = Ranger.useUpdate.useDirect({ params: {} });
  const { update: del } = Ranger.useDelete.useDirect({ params: {} });
  const handleAddChildRange = () => {
    placeLayout(createCreateLayout({ parent: ranges[0].key }));
  };

  const handleSelect: PMenu.MenuProps["onChange"] = {
    rename: () => {
      rename({}, { icon: "Range", name: "Range.Rename" })
        .then((renamed) => {
          if (renamed == null) return;
          update({ ...ranges[0].payload, name: renamed });
        })
        .catch(console.error);
    },
    delete: () => {
      confirm(ranges[0])
        .then((confirmed) => {
          if (confirmed) del(ranges[0].key);
        })
        .catch(console.error);
    },
    addChildRange: handleAddChildRange,
  };

  return (
    <PMenu.Menu level="small" iconSpacing="small" onChange={handleSelect}>
      <PMenu.Item startIcon={<Icon.Add />} itemKey="create">
        Create
      </PMenu.Item>
      {isSingle && <Menu.RenameItem />}
      {!isEmpty && deleteMenuItem}
      {isSingle && addChildRangeMenuItem}
    </PMenu.Menu>
  );
};

export const EXPLORER_LAYOUT: Layout.State = {
  key: EXPLORER_LAYOUT_TYPE,
  windowKey: EXPLORER_LAYOUT_TYPE,
  type: EXPLORER_LAYOUT_TYPE,
  name: "Range Explorer",
  icon: "Explore",
  location: "mosaic",
};

export const Explorer: Layout.Renderer = () => {
  const { data, getItem, subscribe, retrieve } = Ranger.useList({
    sort: (a, b) => compare.stringsWithNumbers(a.stage, b.stage),
  });
  const { fetchMore, search } = List.usePager({ retrieve, pageSize: 100 });
  const [searchTerm, setSearchTerm] = useState("");
  const menuProps = PMenu.useContextMenu();
  const [selected, setSelected] = useState<ranger.Key[]>([]);
  return (
    <Select.Frame<ranger.Key, ranger.Range>
      multiple
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={fetchMore}
      value={selected}
      onChange={setSelected}
    >
      <PMenu.ContextMenu
        menu={(p) => <ContextMenu {...p} ranges={getItem(p.keys)} />}
        {...menuProps}
      />
      <Align.Space>
        <Align.Space x bordered style={{ padding: "2rem" }} background={1}>
          <Input.Text
            size="large"
            level="h4"
            variant="natural"
            value={searchTerm}
            placeholder={
              <Text.WithIcon level="h4" startIcon={<Icon.Search />}>
                Search Ranges...
              </Text.WithIcon>
            }
            onChange={(value) => {
              setSearchTerm(value);
              search(value);
            }}
          />
        </Align.Space>
        <List.Items
          onContextMenu={menuProps.open}
          displayItems={Infinity}
          style={{ height: "100%" }}
        >
          {listItem}
        </List.Items>
      </Align.Space>
    </Select.Frame>
  );
};
