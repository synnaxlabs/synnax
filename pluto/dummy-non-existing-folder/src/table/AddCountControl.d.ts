import "./AddCountControl.css";
import { type ReactElement } from "react";
export interface AddCountControlProps {
    resourceName: "row" | "column";
    onAdd: (count: number) => void;
    className?: string;
}
export declare const AddCountControl: ({ resourceName, onAdd, className, }: AddCountControlProps) => ReactElement;
//# sourceMappingURL=AddCountControl.d.ts.map