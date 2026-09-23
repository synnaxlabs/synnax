import { z } from "zod";
/** Every size step, from smallest to largest. */
export declare const SIZES: readonly ["tiny", "small", "medium", "large", "huge"];
/** Schema for a {@link Size}. */
export declare const size: z.ZodEnum<{
    huge: "huge";
    large: "large";
    medium: "medium";
    small: "small";
    tiny: "tiny";
}>;
/**
 * The shared size scale. Components map it onto their own height, padding, and text
 * level, so a row of components at one size lines up.
 */
export type Size = z.infer<typeof size>;
/** Height in pixels for each {@link Size}. Mirrors the `--pluto-height-*` CSS vars. */
export declare const HEIGHTS: Record<Size, number>;
/** @returns true if the value is one of {@link SIZES}. */
export declare const isSize: (value: unknown) => value is Size;
//# sourceMappingURL=size.d.ts.map