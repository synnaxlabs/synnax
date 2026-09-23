import { type Theming } from "@synnaxlabs/lyra/theming";
import { type optional } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type ContainerProps } from "./Container";
import { type EntriesProps, type EntryData } from "./Entries";
export interface SimpleProps extends Omit<ContainerProps, "value" | "onChange" | "background" | "draggable" | "gap">, Pick<EntriesProps, "onColorChange" | "onLabelChange" | "onVisibleChange"> {
    data?: optional.Optional<EntryData, "visible">[];
    position?: ContainerProps["value"];
    onPositionChange?: ContainerProps["onChange"];
    allowEntryVisibleChange?: boolean;
    background?: Theming.Shade;
}
export declare const Simple: ({ data, onColorChange, onLabelChange, onVisibleChange, position, onPositionChange, allowEntryVisibleChange, background, ...rest }: SimpleProps) => ReactElement | null;
//# sourceMappingURL=Simple.d.ts.map