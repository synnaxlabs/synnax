import { type channel } from "@synnaxlabs/client";
import { type notation } from "@synnaxlabs/x";
import { telem } from "../../telem/aether";
export interface StringSourceArgs {
    channel?: channel.Key;
    rollingAverage?: number;
    precision?: number;
    notation?: notation.Notation;
}
/** stringSource builds the formatted display pipeline for a value channel. */
export declare const stringSource: ({ channel, rollingAverage, precision, notation, }: StringSourceArgs) => telem.StringSourceSpec;
//# sourceMappingURL=telem.d.ts.map