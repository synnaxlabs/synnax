import { type schematic } from "@synnaxlabs/client";
import { type dimensions, type location } from "@synnaxlabs/x";
import { type telem } from "../../../../telem/aether";
import { type Scale as VisScale } from "../../../../vis/scale";
/** Side the ticks and the readout sit on until the user moves them. */
export declare const DEFAULT_SIDE: location.Outer;
/** Stored shape of a live scale indicator, shared by every symbol that renders one. */
export type Config = schematic.ScaleIndicatorConfig;
export declare const DEFAULT_DIMENSIONS: dimensions.Dimensions;
/** source builds the smoothed read pipeline the indicator's value is drawn from. */
export declare const source: ({ channel, rollingAverage }: Config) => telem.NumberSourceSpec;
/** visProps translates the stored hidden flags into the vis scale's show flags. */
export declare const visProps: ({ fillHidden, caretHidden, scaleHidden, ...rest }: Config) => Omit<VisScale.UseProps, "aetherKey" | "box"> & {
    showScale: boolean;
};
//# sourceMappingURL=config.d.ts.map