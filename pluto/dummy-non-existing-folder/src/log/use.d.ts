import { type optional } from "@synnaxlabs/x";
import { type Dispatch, type SetStateAction } from "react";
import { type z } from "zod";
import { Aether } from "../aether";
import { log } from "./aether";
export interface UseProps extends optional.Optional<Omit<z.input<typeof log.logStateZ>, "region" | "scrollPosition" | "scrollback" | "empty" | "scrolling" | "resumedAt" | "wheelPos" | "selectionStart" | "selectionEnd" | "visibleStart" | "selectedText" | "selectedLines" | "computedLineHeight" | "channelNames" | "channelDataTypes">, "visible">, Aether.ComponentProps {
    /** Controlled pause state. When set, the log pauses scrolling while true. */
    hold?: boolean;
    /** Called when the pause state changes from inside the log: a scroll up or the H
     * trigger pauses it, scrolling back to the bottom resumes it. Controlled callers
     * must reflect the value back through hold. */
    onHold?: (hold: boolean) => void;
}
export type LogState = z.output<typeof log.logStateZ>;
export interface UseReturn {
    state: LogState;
    setState: Dispatch<SetStateAction<LogState>>;
}
export declare const use: ({ aetherKey, font, visible, channelNamesHidden, receiptTimestampHidden, timestampPrecision, channels: rawChannels, color, telem, hold, onHold, }: UseProps) => UseReturn;
//# sourceMappingURL=use.d.ts.map