import "./Grouped.css";
import { type ReactElement } from "react";
import { type ContainerProps } from "./Container";
import { type EntriesProps } from "./Entries";
export interface GroupData {
    key: string;
    name: string;
    data: EntriesProps["data"];
}
export interface GroupedProps extends Omit<ContainerProps, "value" | "onChange" | "background" | "draggable" | "gap">, Pick<EntriesProps, "background" | "allowVisibleChange" | "onColorChange" | "onLabelChange" | "onVisibleChange"> {
    data: GroupData[];
    position?: ContainerProps["value"];
    onPositionChange?: ContainerProps["onChange"];
}
export declare const Grouped: ({ data, background, allowVisibleChange, onColorChange, onLabelChange, onVisibleChange, position, onPositionChange, ...rest }: GroupedProps) => ReactElement | null;
//# sourceMappingURL=Grouped.d.ts.map