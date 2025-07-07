import { type record } from "@synnaxlabs/x";
import { type FC } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { type ItemProps } from "@/tree/Tree";

interface BaseProps {
  className: string;
  style: React.CSSProperties;
}
const createItem = (Base: FC<BaseProps>) => {
  const Item = <K extends record.Key>({ index, ...rest }: ItemProps<K>) => {
    const style = { marginLeft: `${index * 2.5 + 1.5}rem` };
    const className = CSS.B("tree-item");
    return <Base style={style} className={className} {...rest} />;
  };
  return Item;
};

export const Item = createItem(Button.Button);
export const ItemLink = createItem(Button.Link);
