import { Flex } from "@synnaxlabs/lyra/flex";
import { type Theming } from "@synnaxlabs/lyra/theming";
import { type color, type optional, type state } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export interface EntryData {
    color: color.Crude;
    key: string;
    label: string;
    visible: boolean;
}
export interface EntriesProps {
    allowVisibleChange?: boolean;
    background?: Theming.Shade;
    data: optional.Optional<EntryData, "visible">[];
    onColorChange?: (key: string, color: color.Color) => void;
    onLabelChange?: (key: string, label: string) => void;
    onVisibleChange?: (key: string, visible: boolean) => void;
    colorPickerVisible?: boolean;
    onColorPickerVisibleChange?: state.Setter<boolean>;
    entryProps?: Omit<Flex.BoxProps, "background">;
}
export declare const Entries: import("react").MemoExoticComponent<({ data, allowVisibleChange, background, entryProps, ...rest }: EntriesProps) => ReactElement>;
//# sourceMappingURL=Entries.d.ts.map