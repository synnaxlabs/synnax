import { RefObject, useEffect } from "react";

export const useClickoutside = (
  ref: RefObject<HTMLElement>,
  onClickOutside: () => void
): void =>
  useEffect(() => {
    const { current: el } = ref;
    const handleClickOutside = ({ target }: MouseEvent): void => {
      if (el != null && !el.contains(target as Node)) onClickOutside();
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [ref, onClickOutside]);
