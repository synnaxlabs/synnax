import {
  Dispatch,
  FocusEventHandler,
  forwardRef,
  ReactElement,
  RefObject,
  SetStateAction,
  useCallback,
  useRef,
  useState,
} from "react";

import clsx from "clsx";

import { Space, SpaceProps } from "../Space";

import { useClickOutside } from "@/hooks";
import { visibleCls } from "@/util/css";
import { VerticalLocation } from "@/util/spatial";

import "./Dropdown.css";

export interface UseDropdownReturn {
  visible: boolean;
  setVisible: Dispatch<SetStateAction<boolean>>;
  ref: RefObject<HTMLDivElement>;
  onFocus: FocusEventHandler;
}
export const useDropdown = (): UseDropdownReturn => {
  const [visible, setVisible] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useClickOutside(ref, () => setVisible(false));
  const onFocus = useCallback(() => setVisible(true), []);
  return { visible, ref, onFocus, setVisible };
};

export interface DropdownProps
  extends Omit<UseDropdownReturn, "onFocus" | "setVisible">,
    Omit<SpaceProps, "ref" | "reverse"> {
  location?: VerticalLocation;
  children: [
    ReactElement<{ onFocus: FocusEventHandler; autoFocus: boolean }>,
    JSX.Element
  ];
}

export const Dropdown = forwardRef<HTMLDivElement, DropdownProps>(
  (
    { visible, children, location = "top", ...props }: DropdownProps,
    ref
  ): JSX.Element => (
    <Space
      {...props}
      ref={ref}
      className="pluto-dropdown__container"
      reverse={location === "top"}
    >
      {children[0]}
      <Space
        className={clsx(
          "pluto-dropdown__dialog",
          `pluto-dropdown__dialog--${location}`,
          `pluto-dropdown__dialog--${visibleCls(visible)}`
        )}
        empty
      >
        {children[1]}
      </Space>
    </Space>
  )
);

Dropdown.displayName = "Dropdown";
