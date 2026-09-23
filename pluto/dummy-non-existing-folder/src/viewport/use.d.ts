import { Triggers } from "@synnaxlabs/lyra/triggers";
import { box, dimensions, xy } from "@synnaxlabs/x";
import { type ForwardedRef, type RefObject } from "react";
import { z } from "zod";
export interface UseEvent {
    box: box.Box;
    cursor: xy.XY;
    mode: Mode;
    stage: Triggers.Stage;
}
export type UseHandler = (e: UseEvent) => void;
export type UseTriggers = Triggers.ModeConfig<TriggerMode>;
export interface UseRefValue {
    reset: () => void;
}
export interface UseProps {
    triggers?: UseTriggers;
    onChange?: UseHandler;
    resetOnDoubleClick?: boolean;
    threshold?: dimensions.Dimensions;
    initial?: box.Box;
    ref?: RefObject<UseRefValue | undefined> | ForwardedRef<UseRefValue | undefined>;
}
export interface UseReturn {
    mode: Mode;
    maskBox: box.Box;
    ref: React.RefObject<HTMLDivElement | null>;
}
type StringLiteral<T> = T extends string ? (string extends T ? never : T) : never;
declare const TRIGGER_MODES: readonly ["zoom", "pan", "select", "zoomReset", "cancel"];
export declare const MODES: readonly ["zoom", "pan", "select", "zoomReset", "cancel", "click"];
export declare const modeZ: z.ZodEnum<{
    cancel: "cancel";
    click: "click";
    pan: "pan";
    select: "select";
    zoom: "zoom";
    zoomReset: "zoomReset";
}>;
export type Mode = StringLiteral<(typeof MODES)[number]>;
type TriggerMode = StringLiteral<(typeof TRIGGER_MODES)[number]>;
export declare const MASK_MODES: Mode[];
export declare const ZOOM_DEFAULT_TRIGGERS: UseTriggers;
export declare const PAN_DEFAULT_TRIGGERS: UseTriggers;
export declare const SELECT_DEFAULT_TRIGGERS: UseTriggers;
export declare const DEFAULT_TRIGGERS: Record<Mode, UseTriggers>;
export declare const use: ({ onChange, triggers: initialTriggers, initial, threshold: threshold_, ref, }: UseProps) => UseReturn;
export {};
//# sourceMappingURL=use.d.ts.map