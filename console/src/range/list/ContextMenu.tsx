import { type ranger } from "@synnaxlabs/client";
import { Divider, Form, type List, Menu as PMenu, Ranger } from "@synnaxlabs/pluto";

import { Menu } from "@/components";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";
import { createChildRangeMenuItem, deleteMenuItem } from "@/range/ContextMenu";
import { createCreateLayout } from "@/range/Create";

export interface ContextMenuProps extends PMenu.ContextMenuMenuProps {
  getItem: List.GetItem<string, ranger.Range>;
}

export const ContextMenu = ({ keys, getItem }: ContextMenuProps) => {
  const ranges = getItem(keys);
  const isEmpty = ranges.length === 0;
  const isSingle = ranges.length === 1;
  const placeLayout = Layout.usePlacer();
  const ctx = Form.useContext();
  const rename = Modals.useRename();
  const confirm = useConfirmDelete({
    type: "Range",
    description: "Deleting this range will also delete all child ranges.",
  });
  const { update: del } = Ranger.useDelete.useDirect({ params: {} });
  const handleAddChildRange = () => {
    placeLayout(createCreateLayout({ parent: ranges[0].key }));
  };

  const handleSelect: PMenu.MenuProps["onChange"] = {
    rename: () => {
      rename({ initialValue: ranges[0].name }, { icon: "Range", name: "Range.Rename" })
        .then((renamed) => {
          if (renamed == null) return;
          ctx.set("name", renamed);
        })
        .catch(console.error);
    },
    delete: () => {
      confirm(ranges)
        .then((confirmed) => {
          if (confirmed) del(ranges.map((r) => r.key));
        })
        .catch(console.error);
    },
    addChildRange: handleAddChildRange,
  };

  return (
    <PMenu.Menu level="small" gap="small" onChange={handleSelect}>
      {isSingle && <Menu.RenameItem />}
      {!isEmpty && deleteMenuItem}
      <Divider.Divider x />
      {isSingle && createChildRangeMenuItem}
    </PMenu.Menu>
  );
};
