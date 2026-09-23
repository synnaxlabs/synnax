import { type ReactElement } from "react";
export declare const reactElementToArray: <P = any, T extends string | React.JSXElementConstructor<any> = string | React.JSXElementConstructor<any>>(children: ReactElement<P, T> | Array<ReactElement<P, T>>) => Array<ReactElement<P, T>>;
export declare const isValidElement: <P = any, T extends string | React.JSXElementConstructor<any> = string | React.JSXElementConstructor<any>>(child: unknown) => child is ReactElement<P, T>;
//# sourceMappingURL=children.d.ts.map