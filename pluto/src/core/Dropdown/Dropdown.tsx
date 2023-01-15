// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

import { Space, SpaceProps } from "@/core/Space";
import { useClickOutside } from "@/hooks";
import { VerticalLocation } from "@/spatial";
import { visibleCls } from "@/util/css";

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
    Omit<SpaceProps, "ref" | "reverse" | "size" | "empty"> {
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
      empty
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
