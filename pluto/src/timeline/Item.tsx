import { type bounds } from "@synnaxlabs/x";
import { type ReactElement, useEffect } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { useContext } from "@/timeline/context";

export interface ItemProps<T extends number | bigint = number> extends Flex.BoxProps {
  itemKey: string;
  bounds: bounds.Bounds<T>;
  onBoundsChange: (bounds: bounds.Bounds<T>) => void;
}

export const Item = <T extends number | bigint = number>({
  itemKey,
  bounds,
  onBoundsChange,
  className,
  ...rest
}: ItemProps<T>): ReactElement => {
  const { setEntry } = useContext("Timeline.Item");
  useEffect(() => {
    setEntry({ key: itemKey, bounds });
  }, [bounds, onBoundsChange]);
  return <Flex.Box className={CSS(CSS.BE("timeline", "item"), className)} {...rest} />;
};
