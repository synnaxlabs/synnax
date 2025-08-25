import { type effect } from "@synnaxlabs/client";
import { Effect, Form, Icon, type List, Menu as PMenu } from "@synnaxlabs/pluto";

import { Menu } from "@/components";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";

export interface ContextMenuProps extends PMenu.ContextMenuMenuProps {
  getItem: List.GetItem<string, effect.Effect>;
}

export const ContextMenu = ({ keys, getItem }: ContextMenuProps) => {
  const effects = getItem(keys);
  const isEmpty = effects.length === 0;
  const isSingle = effects.length === 1;
  const ctx = Form.useContext();
  const rename = Modals.useRename();
  const confirm = useConfirmDelete({
    type: "Effect",
    description: "Deleting this effect will permanently remove it.",
  });
  const { update: del } = Effect.useDelete.useDirect({
    params: { keys },
  });

  const handleSelect: PMenu.MenuProps["onChange"] = {
    rename: () => {
      rename(
        { initialValue: effects[0].name },
        { icon: "Effect", name: "Effect.Rename" },
      )
        .then((renamed) => {
          if (renamed == null) return;
          ctx.set("name", renamed);
        })
        .catch(console.error);
    },
    delete: () => {
      confirm(effects)
        .then((confirmed) => {
          if (confirmed) del();
        })
        .catch(console.error);
    },
  };

  return (
    <PMenu.Menu level="small" gap="small" onChange={handleSelect}>
      {isSingle && <Menu.RenameItem />}
      {!isEmpty && (
        <PMenu.Item itemKey="delete">
          <Icon.Delete />
          Delete
        </PMenu.Item>
      )}
    </PMenu.Menu>
  );
};
