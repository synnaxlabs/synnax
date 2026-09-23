import { type channel } from "@synnaxlabs/client";
import { type bounds } from "@synnaxlabs/x";
import { telem } from "../../../../telem/aether";
export declare const DEFAULT_THRESHOLD: bounds.Bounds;
/** booleanSource builds the boolean read pipeline for a state channel. */
export declare const booleanSource: (channel?: channel.Key, threshold?: bounds.Bounds) => telem.BooleanSourceSpec;
/** booleanSink builds the boolean command pipeline for a command channel. */
export declare const booleanSink: (channel?: channel.Key) => telem.BooleanSinkSpec;
/** numberSink builds the numeric command pipeline for a command channel. */
export declare const numberSink: (channel?: channel.Key) => telem.NumberSinkSpec;
/** stringSink builds the string command pipeline for a command channel. */
export declare const stringSink: (channel?: channel.Key) => telem.StringSinkSpec;
/** numberSource builds the numeric read pipeline for a state channel. */
export declare const numberSource: (channel?: channel.Key) => telem.NumberSourceSpec;
export interface SmoothedNumberSourceArgs {
    channel?: channel.Key;
    rollingAverage?: number;
}
/** smoothedNumberSource builds the rolling-average read pipeline for a channel. */
export declare const smoothedNumberSource: ({ channel, rollingAverage, }: SmoothedNumberSourceArgs) => telem.NumberSourceSpec;
export interface ControlChipArgs {
    channel?: channel.Key;
    authority?: number;
}
/** chipSink builds the control acquisition sink for a command channel. */
export declare const chipSink: ({ channel, authority, }: ControlChipArgs) => telem.BooleanSinkSpec;
//# sourceMappingURL=telem.d.ts.map