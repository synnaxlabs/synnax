import { RefObject, useEffect } from "react";

export const useClickoutside = (
  ref: RefObject<HTMLElement>,
  onClickOutside: () => void
) => {
  useEffect(() => {
    const { current: el } = ref;
    const handleClickOutside = ({ target }: MouseEvent) => {
      if (el && !el.contains(target as Node)) onClickOutside();
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [ref, onClickOutside]);
};
