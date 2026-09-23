import { type ComponentPropsWithRef, type ReactElement } from "react";
import { type OrientableProps } from "./orientable";
export interface DivProps extends Omit<ComponentPropsWithRef<"div">, "color" | "onResize">, OrientableProps {
}
export declare const Div: ({ className, ...rest }: DivProps) => ReactElement;
//# sourceMappingURL=Div.d.ts.map