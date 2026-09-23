import { color } from "@synnaxlabs/x";
/**
 * Creates a set of CSS variables representing different opacities of a given color.
 * @param prefix The prefix to use for the CSS variable names.
 * @param opacities A list of the opacities to create
 * @returns Record mapping the CSS variable names to their values.
 */
export declare const createHexOpacityVariants: (prefix: string, hex: color.Crude, opacities: readonly number[]) => Record<string, string>;
//# sourceMappingURL=color.d.ts.map