import "./Text.css";
import { type text } from "@synnaxlabs/x";
import { type ComponentPropsWithoutRef, type ReactElement, type ReactNode } from "react";
import { Flex } from "../flex";
import { type Generic } from "../generic";
import { type Status } from "../status";
type AnchorProps = ComponentPropsWithoutRef<"a">;
/** The typeface and treatment: body prose, monospace code, a key cap, or a link. */
export type Variant = "prose" | "code" | "keyboard" | "link";
/** What happens to text that outruns its box. */
export type Overflow = "ellipsis" | "fade" | "clip" | "nowrap" | "wrap";
/**
 * The text-specific props {@link TextProps} adds. Text is a flex box, so it also takes
 * every {@link Flex.BoxExtensionProps} for laying out icons beside a label.
 */
export interface ExtensionProps extends Flex.BoxExtensionProps, Pick<AnchorProps, "href" | "target" | "rel"> {
    /** The type scale step, which also picks the rendered element: p, h1, h2. */
    level?: text.Level;
    children?: ReactNode;
    /** The font weight. Defaults to the level's own. */
    weight?: text.Weight;
    /** The typeface treatment. Defaults to "prose". */
    variant?: Variant;
    /** Prefixes a scheme-less href with `https://`. */
    autoFormatHref?: boolean;
    /** The element to render when `level` does not imply one. */
    defaultEl?: Generic.ElementType;
    /** What to do with text that outruns its box. */
    overflow?: Overflow;
    /** Tints the text with a status color. */
    status?: Status.Variant;
    /** Truncates past this many lines. Implies an ellipsis. */
    lineClamp?: number;
}
/** Props for {@link Text}, on top of the props of the element it renders as. */
export type TextProps<E extends Generic.ElementType = "p"> = Omit<Generic.OptionalElementProps<E>, "color"> & ExtensionProps;
/**
 * @returns true if the children are a lone icon, which the chassis renders in a square
 * rather than a padded pill.
 */
export declare const isSquare: (children: ReactNode) => boolean;
/** Resolves which element a text-based component renders as. */
export declare const parseElement: <E extends Generic.ElementType = "p">(level?: text.Level, el?: E, defaultEl?: Generic.ElementType, variant?: Variant, href?: string) => E | undefined;
/**
 * Renders text at a step on the shared type scale. It is also a flex box, so icons and
 * a label lay out inside one element with no wrapper.
 *
 * @example <Text.Text level="h3">Ranges</Text.Text>
 * @example <Text.Text level="p" overflow="ellipsis"><Icon.Range />{name}</Text.Text>
 */
export declare const Text: <E extends Generic.ElementType = "p">({ level, className, style, weight, defaultEl, el, variant, overflow, href, autoFormatHref, status, lineClamp, ...rest }: TextProps<E>) => ReactElement;
export {};
//# sourceMappingURL=Text.d.ts.map