import { Icon, Menu as Core } from "@synnaxlabs/pluto";

export interface MenuProps {
  itemKey: string;
  path: string;
  onDelete: (path: string) => void;
}

export const Menu = ({ itemKey, onDelete }: MenuProps) => (
  <Core.Menu level="small" onChange={{ delete: () => onDelete(itemKey) }}>
    <Core.Item itemKey="delete">
      <Icon.Delete />
      Delete
    </Core.Item>
  </Core.Menu>
);
