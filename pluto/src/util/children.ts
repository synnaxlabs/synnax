import { Children, ReactElement } from "react";

export const reactElementToArray = <
  P = any,
  T extends string | React.JSXElementConstructor<any> =
    | string
    | React.JSXElementConstructor<any>
>(
  children: ReactElement<P, T> | Array<ReactElement<P, T>>
): Array<ReactElement<P, T>> => Children.toArray(children) as Array<ReactElement<P, T>>;
