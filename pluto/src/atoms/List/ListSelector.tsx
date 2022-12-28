import { useEffect } from "react";

import { useListContext } from "./ListContext";

import { useSelectMultiple, UseSelectMultipleProps } from "@/hooks/useSelectMultiple";
import { RenderableRecord } from "@/util/record";

export interface ListSelectorProps<E extends RenderableRecord<E>>
  extends Omit<UseSelectMultipleProps<E>, "data"> {}

export const ListSelector = <E extends RenderableRecord<E>>({
  value,
  ...props
}: ListSelectorProps<E>): null => {
  const {
    data,
    setTransform,
    deleteTransform,
    select: { setOnSelect, setClear },
  } = useListContext();

  const { onSelect, transform, clear } = useSelectMultiple({ data, value, ...props });

  useEffect(() => {
    setOnSelect(() => onSelect);
    setClear(() => clear);
  }, [onSelect, clear]);

  useEffect(() => {
    if (value == null || value.length === 0) deleteTransform("select");
    setTransform("select", transform);
  }, [transform]);

  return null;
};
