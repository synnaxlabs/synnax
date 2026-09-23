import { type ReactElement } from "react";
/**
 * A function child: the parent calls it with the props of the thing being rendered,
 * so the caller decides the markup while the parent keeps the data and the loop.
 */
export type RenderProp<P extends Record<string, any>, R = ReactElement | null> = (props: P) => R;
/** Component prop takes in a component and turns it into a render prop. */
export declare const renderProp: <P extends Record<string, any>, R = ReactElement | null>(Component: React.ComponentType<P>) => RenderProp<P, R>;
/** @returns true if children is a {@link RenderProp} rather than a rendered node. */
export declare const isRenderProp: <P extends Record<string, any>>(children: React.ReactNode | RenderProp<P>) => children is RenderProp<P>;
//# sourceMappingURL=renderProp.d.ts.map