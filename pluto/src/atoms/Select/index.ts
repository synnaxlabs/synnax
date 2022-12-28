import { Select as CoreSelect } from "./Select";
import { SelectMultiple } from "./SelectMultiple";

export type { SelectMultipleProps } from "./SelectMultiple";
export type { SelectProps } from "./Select";

type CoreSelectType = typeof CoreSelect;

interface SelectType extends CoreSelectType {
  Multiple: typeof SelectMultiple;
}

export const Select = CoreSelect as SelectType;

Select.Multiple = SelectMultiple;
