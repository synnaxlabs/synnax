import { type ComponentPropsWithRef, type JSX, type ReactElement } from "react";
/** The name of any built-in HTML element. */
export type ElementType = keyof JSX.IntrinsicElements;
/** Props common to every element, for a component that has not chosen one yet. */
export type ElementPropsWithoutEl = ComponentPropsWithRef<ElementType>;
/** The props of element E, plus the `el` prop naming it. */
export type ElementProps<E extends keyof JSX.IntrinsicElements> = {
    el: E;
} & ComponentPropsWithRef<E>;
/** {@link ElementProps} where the component supplies a default element. */
export type OptionalElementProps<E extends keyof JSX.IntrinsicElements> = {
    el?: E;
} & ComponentPropsWithRef<E>;
/**
 * Renders the HTML element named by `el`, typed to that element's own props. It backs
 * every component that lets the caller pick its tag.
 */
export declare const Element: <E extends keyof JSX.IntrinsicElements>({ el, children, ...rest }: ElementProps<E>) => ReactElement;
//# sourceMappingURL=Generic.d.ts.map