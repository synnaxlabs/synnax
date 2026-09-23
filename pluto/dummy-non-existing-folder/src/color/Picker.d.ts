import "./Picker.css";
import { type Input } from "@synnaxlabs/lyra/input";
import { color } from "@synnaxlabs/x";
import { type ComponentPropsWithoutRef, type ReactElement } from "react";
export interface PickerProps extends Input.Control<color.Crude, color.Color>, Omit<ComponentPropsWithoutRef<"div">, "onChange"> {
    onDelete?: () => void;
    position?: number;
}
export declare const Picker: ({ value, onChange, position, onDelete, ...rest }: PickerProps) => ReactElement;
//# sourceMappingURL=Picker.d.ts.map