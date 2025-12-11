import { Icon, Menu as Core } from "@synnaxlabs/pluto";

export interface MenuProps {
  path: string;
  onDelete: (path: string) => void;
}

export const Menu = ({ path, onDelete }: MenuProps) => {
  const handleSelect: Core.MenuProps["onChange"] = {
    delete: () => onDelete(path),
  };
  return (
    <Core.Menu level="small" onChange={handleSelect}>
      <Core.Item itemKey="delete">
        <Icon.Delete />
        Delete
      </Core.Item>
    </Core.Menu>
  );
};
