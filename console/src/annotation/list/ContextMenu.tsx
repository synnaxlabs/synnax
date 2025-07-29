import { Icon, Menu as PMenu } from "@synnaxlabs/pluto";

import { Menu } from "@/components";

export interface ContextMenuProps {
  onEdit: () => void;
  onDelete: () => void;
}
export const ContextMenu = ({ onEdit, onDelete }: ContextMenuProps) => {
  const handleSelect: PMenu.MenuProps["onChange"] = {
    edit: onEdit,
    delete: onDelete,
  };

  return (
    <PMenu.Menu level="small" gap="small" onChange={handleSelect}>
      <PMenu.Item itemKey="edit" startIcon={<Icon.Edit />}>
        Edit
      </PMenu.Item>
      <PMenu.Item itemKey="delete" startIcon={<Icon.Delete />}>
        Delete
      </PMenu.Item>
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};
