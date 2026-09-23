import { type channel, schematic } from "@synnaxlabs/client";
export declare const stateConfigZ: import("zod").ZodObject<{
    authority: import("zod").ZodOptional<import("zod").ZodInt>;
    hidden: import("zod").ZodDefault<import("zod").ZodBoolean>;
    chipHidden: import("zod").ZodDefault<import("zod").ZodBoolean>;
    indicatorHidden: import("zod").ZodDefault<import("zod").ZodBoolean>;
    orientation: import("zod").ZodDefault<import("zod").ZodEnum<{
        bottom: "bottom";
        center: "center";
        left: "left";
        right: "right";
        top: "top";
    }>>;
}, import("zod/v4/core").$strip>;
export type StateConfig = schematic.ControlStateConfig;
/** reveal clears every hidden flag, so the control state shows once a command channel
 * is chosen. */
export declare const reveal: (config?: StateConfig) => StateConfig;
export interface State {
    config?: StateConfig;
    channel?: channel.Key;
    onChange?: (next: {
        control: StateConfig;
    }) => void;
}
export declare const State: import("react").FC<State>;
//# sourceMappingURL=State.d.ts.map