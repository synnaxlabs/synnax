import "./Viewport.css";
import { type PropsWithChildren, type ReactElement } from "react";
import { Viewport as Base } from "../viewport";
export interface ViewportProps extends PropsWithChildren, Base.UseProps {
}
export declare const selectViewportEl: (el: HTMLElement | null) => Element | null;
export declare const Viewport: ({ ref, children, initial, onChange, ...rest }: ViewportProps) => ReactElement;
//# sourceMappingURL=Viewport.d.ts.map