import { Dropdown as CoreDropdown, useDropdown } from "./Dropdown";
export type { DropdownProps } from "./Dropdown";

type CoreDropdownType = typeof CoreDropdown;

interface DropdownType extends CoreDropdownType {
  use: typeof useDropdown;
}

export const Dropdown = CoreDropdown as DropdownType;

Dropdown.use = useDropdown;
