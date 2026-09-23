import "./Breadcrumb.css";
import { type ReactElement, type ReactNode } from "react";
import { type Generic } from "../generic";
import { Text } from "../text";
/** Which segments of a breadcrumb are drawn in the full text color. */
export type HighlightVariant = "last" | "first" | "all";
/** Props for {@link Breadcrumb}. */
export type BreadcrumbProps<E extends Generic.ElementType = "p"> = Omit<Text.TextProps<E>, "children"> & {
    children: ReactNode;
    highlightVariant?: HighlightVariant;
};
/** Props for {@link Segment}. */
export type SegmentProps<E extends Generic.ElementType = "span"> = Text.TextProps<E>;
/** One step of a {@link Breadcrumb}. A plain string child becomes one on its own. */
export declare const Segment: <E extends Generic.ElementType = "span">({ children, ...rest }: SegmentProps<E>) => ReactElement;
/**
 * A path rendered as segments with carets between them.
 *
 * @example <Breadcrumb.Breadcrumb highlightVariant="last">{parts}</Breadcrumb.Breadcrumb>
 */
export declare const Breadcrumb: ({ children, highlightVariant, ...rest }: BreadcrumbProps) => ReactElement;
interface MapURLSegmentsProps {
    href: string;
    segment: string;
    index: number;
}
/** Renders one element per path segment, each given the URL up to that point. */
export declare const mapURLSegments: (url: string, callback: (props: MapURLSegmentsProps) => ReactElement) => ReactElement<unknown, string | import("react").JSXElementConstructor<any>>[];
export {};
//# sourceMappingURL=Breadcrumb.d.ts.map