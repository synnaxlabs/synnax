import "./Log.css";
import { type Flex } from "@synnaxlabs/lyra/flex";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement, type ReactNode } from "react";
import { type UseProps } from "./use";
export declare const PAUSE_TRIGGER: Triggers.Trigger;
export interface BaseProps extends UseProps, Omit<Flex.BoxProps, "color"> {
    emptyContent?: ReactElement;
    extraContextMenuItems?: ReactNode;
    /** When set, the context menu offers undo and redo for the host document. */
    undoRedo?: Menu.UndoRedoItemsProps;
    enableTriggers?: Triggers.Condition;
}
export declare const Base: ({ aetherKey, font, className, visible, channelNamesHidden, receiptTimestampHidden, timestampPrecision, channels, emptyContent, color, telem, extraContextMenuItems, undoRedo, enableTriggers, hold, onHold, children, ...rest }: BaseProps) => ReactElement | null;
//# sourceMappingURL=Base.d.ts.map