import { type record } from "@synnaxlabs/x";
import { type FC } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Select } from "@/select";
import { type ItemProps } from "@/tree/Tree";

interface BaseProps {
  className: string;
  style: React.CSSProperties;
}
const createItem = (Base: FC<BaseProps>) => {
  const Item = <K extends record.Key>({ depth, itemKey, ...rest }: ItemProps<K>) => {
    const style = { marginLeft: `${depth * 2.5 + 1.5}rem` };
    const className = CSS.B("tree-item");
    const { onSelect } = Select.useItemState<K>(itemKey);
    return (
      <Base
        variant="text"
        style={style}
        className={className}
        {...rest}
        onClick={onSelect}
      />
    );
  };
  return Item;
};

export const Item = createItem(Button.Button);
export const ItemLink = createItem(Button.Link);
