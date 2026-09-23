import "./Box.css";
import { type color, direction } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import z from "zod";
import { type Component } from "../component";
import { Generic } from "../generic";
import { type Theming } from "../theming";
/** All possible alignments for the cross axis of a space */
export declare const ALIGNMENTS: readonly ["start", "center", "end", "stretch"];
/** Zod schema for {@link Alignment}. */
export declare const alignmentZ: z.ZodEnum<{
    center: "center";
    end: "end";
    start: "start";
    stretch: "stretch";
}>;
/** The alignments for the cross axis of a space */
export type Alignment = z.infer<typeof alignmentZ>;
/** All possible justifications for the main axis of a space */
export declare const JUSTIFICATIONS: readonly ["start", "center", "end", "between", "around", "evenly"];
/** Zod schema for {@link Justification}. */
export declare const justificationZ: z.ZodEnum<{
    around: "around";
    between: "between";
    center: "center";
    end: "end";
    evenly: "evenly";
    start: "start";
}>;
/** The justification for the main axis of a space */
export type Justification = z.infer<typeof justificationZ>;
/**
 * Props for the Box component. Extends generic element props with flex layout
 * capabilities.
 *
 * @example
 * ```tsx
 * // Basic vertical layout with spacing
 * <Box gap="medium">
 *   <div>Item 1</div>
 *   <div>Item 2</div>
 * </Box>
 *
 * // Horizontal centered layout with border
 * <Box x center bordered rounded>
 *   <Button>Action</Button>
 *   <Button>Cancel</Button>
 * </Box>
 *
 * // Full-width container with custom styling
 * <Box full background={11} gap={1.5}>
 *   <Text.H3>Title</Text.H3>
 *   <Text.P>Content goes here</Text.P>
 * </Box>
 * ```
 */
export type BoxProps<E extends Generic.ElementType = "div"> = Omit<Generic.OptionalElementProps<E>, "color"> & BoxExtensionProps;
/**
 * Extension props that provide flex layout capabilities to the Box component.
 * These props control layout direction, alignment, spacing, styling, and sizing.
 */
export interface BoxExtensionProps {
    /** Whether to show a border around the container */
    bordered?: boolean;
    /** Border color using Theming.Shade, color.Crude, or false to hide */
    borderColor?: Theming.Shade | color.Crude | false;
    /** Border width in pixels */
    borderWidth?: number;
    /** Border radius. true for the theme default, a Component.Size for a radius
     * scale step, or a number for a specific rem value. Defaults to true when packed. */
    rounded?: boolean | number | Component.Size;
    /** Whether to remove border radius (sharp corners) */
    sharp?: boolean;
    /** Background color using Theming.Shade values */
    background?: Theming.Shade | color.Crude | false;
    /** Whether the container has no visible spacing/padding */
    empty?: boolean;
    /** Spacing between children. Can be a Component.Size ('small', 'medium', 'large') or
     * a number (in rem units) */
    gap?: Component.Size | number;
    /** The flex direction of the container. Defaults to 'y' (column). Can be 'x', 'y',
     * 'left', 'right', 'top', 'bottom' */
    direction?: direction.Crude;
    /** Shorthand for setting direction to 'x' (row). Overrides direction */
    x?: boolean;
    /** Shorthand for setting direction to 'y' (column). Overrides direction if x is
     * not set */
    y?: boolean;
    /** Whether to reverse the direction of children */
    reverse?: boolean;
    /** Main-axis justification of children. See {@link Justification} for options:
     * 'start', 'center', 'end', 'between', 'around', 'evenly' */
    justify?: Justification;
    /** Cross-axis alignment of children. See {@link Alignment} for options: 'start',
     * 'center', 'end', 'stretch' */
    align?: Alignment;
    /** Cross-axis alignment of this box within its own parent flex container,
     * overriding the parent's align. See {@link Alignment} for options: 'start',
     * 'center', 'end', 'stretch' */
    alignSelf?: Alignment;
    /** Flex grow behavior. true sets flex-grow: 1, false leaves unset, number sets
     * specific flex-grow value */
    grow?: boolean | number;
    /** Flex shrink behavior. true sets flex-shrink: 1, false leaves unset, number
     * sets specific flex-shrink value */
    shrink?: boolean | number;
    /** Shorthand for centering both axes (align: 'center', justify: 'center') */
    center?: boolean;
    /** Whether the container should take full width/height. true for both, or specify
     * a direction ('x', 'y') for single axis */
    full?: boolean | direction.Direction;
    /** Whether children should wrap to new lines when they overflow */
    wrap?: boolean;
    /** Height of the container using Component.Size values */
    size?: Component.Size;
    /**
     * Whether to use horizontal layout optimized for packing items into
     * visually consistent groups.
     */
    pack?: boolean;
    /** Text color using Theming.Shade, color.Crude, or false to inherit */
    color?: Theming.Shade | color.Crude | false;
    /** Whether the container should maintain square aspect ratio */
    square?: boolean;
}
/**
 * @returns whether a box laid out along the given direction reverses its children. An
 * explicit `reverse` wins; otherwise "right" and "bottom" reverse.
 */
export declare const shouldReverse: (direction?: direction.Crude, reverse?: boolean) => boolean;
/**
 * Resolves the {@link Box} direction shorthands into a single axis, in the order `x`,
 * `y`, `direction`, then `pack` (which implies "x").
 *
 * @returns undefined when the caller set none of them, leaving the CSS default.
 */
export declare const parseDirection: (dir?: direction.Crude, x?: boolean, y?: boolean, pack?: boolean) => direction.Direction | undefined;
/**
 * The layout primitive. Lays its children out with flexbox and carries the shared
 * border, background, radius, and gap scales, so a caller rarely needs custom CSS.
 *
 * @example <Flex.Box x gap="small" align="center">{children}</Flex.Box>
 * @example <Flex.Box y grow bordered rounded background={1}>{children}</Flex.Box>
 */
export declare const Box: <E extends Generic.ElementType = "div">({ style, align, className, grow, shrink, gap, color, justify, alignSelf, reverse, empty, pack, wrap, center, direction: crudeDirection, rounded, sharp, borderWidth, borderColor, full, background, bordered, x, y, size, square, ...rest }: BoxProps<E>) => ReactElement;
//# sourceMappingURL=Box.d.ts.map