import "./Icon.css";
import { color } from "@synnaxlabs/x";
import { type FC as ReactFC, type ReactElement as BaseReactElement, type Ref } from "react";
import { type IconBaseProps } from "react-icons";
import { type Theming } from "../theming";
/** Props for every icon in the set. */
export interface IconProps extends Omit<IconBaseProps, "color" | "children"> {
    ref?: Ref<SVGSVGElement>;
    /** Fill color. A number selects that step on the theme gray scale. */
    color?: color.Crude | Theming.Shade;
}
/**
 * A type representing a React element that renders an icon. This element encapsulates
 * the visual representation of an icon component, typically rendered as an SVG with the
 * specified base properties.
 */
export interface ReactElement extends BaseReactElement<IconProps> {
}
/** An icon component. Every member of the icon set has this type. */
export interface FC extends ReactFC<IconProps> {
}
interface WrapIconOpts {
    className?: string;
}
/** A raw SVG component, as generated from an icon file. Wrap it with {@link wrapSVGIcon}. */
export interface SVGFC extends ReactFC<IconBaseProps> {
}
/**
 * Turns a raw SVG component into an {@link FC}, applying the icon class names and
 * theme-aware color parsing. Icons are decorative by default: without an explicit
 * `aria-label` they are hidden from the accessibility tree, so they never leak into
 * an ancestor's accessible name.
 */
export declare const wrapSVGIcon: (Base: SVGFC, name: string, { className }?: WrapIconOpts) => FC;
/**
 * createStacked returns an icon that draws Base twice in oblique projection: a back
 * copy shifted up and right, a front copy shifted down and left. A halo painted in the
 * surface color sits under the front copy so the two silhouettes stay separable.
 */
export declare const createStacked: (Base: FC) => FC;
/**
 * @returns an icon that draws Base with a smaller icon badged into any of its four
 * corners. Base itself is returned when no corner is given.
 *
 * @example createComposite(Icon.Channel, { bottomRight: Icon.Add })
 */
export declare const createComposite: (Base: FC, { topRight, topLeft, bottomLeft, bottomRight }: Record<string, FC | undefined>) => FC;
export {};
//# sourceMappingURL=Icon.d.ts.map