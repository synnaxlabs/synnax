import { RefObject, useEffect } from "react";

export const useClickoutside = (
  ref: RefObject<any>,
  onClickOutside: () => void
) => {
  useEffect(() => {
    const { current: el } = ref;
    const handleClickOutside = ({ target }: MouseEvent) => {
      if (el && !el.contains(target)) onClickOutside();
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [ref, onClickOutside]);
};
