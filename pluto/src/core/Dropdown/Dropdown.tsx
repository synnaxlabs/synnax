// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, RefObject, useCallback, useRef, useState } from "react";

import { YLocation } from "@synnaxlabs/x";

import { Pack } from "../Pack";

import { Space, SpaceProps } from "@/core/Space";
import { CSS } from "@/css";
import { useClickOutside } from "@/hooks";

import "./Dropdown.css";

/** Return type for the {@link useDropdown} hook. */
export interface UseDropdownReturn {
  visible: boolean;
  ref: RefObject<HTMLDivElement>;
  close: () => void;
  open: () => void;
  toggle: (vis?: boolean) => void;
}

export const useDropdown = (initialVisible: boolean = false): UseDropdownReturn => {
  const [visible, setVisible] = useState(initialVisible);
  const ref = useRef<HTMLDivElement>(null);
  const toggle = useCallback(
    (vis?: boolean) => setVisible(vis ?? !visible),
    [setVisible]
  );
  const open = useCallback(() => toggle(true), [toggle]);
  const close = useCallback(() => toggle(false), [toggle]);
  useClickOutside(ref, close);
  return { visible, ref, open, close, toggle };
};

/** Props for the {@link Dropdown} component. */
export interface DropdownProps
  extends Pick<UseDropdownReturn, "visible">,
    Omit<SpaceProps, "ref" | "reverse" | "size" | "empty"> {
  location?: YLocation;
  children: [JSX.Element, JSX.Element];
  keepMounted?: boolean;
}

export const Dropdown = forwardRef<HTMLDivElement, DropdownProps>(
  (
    {
      visible,
      children,
      location = "bottom",
      keepMounted = true,
      ...props
    }: DropdownProps,
    ref
  ): JSX.Element => (
    <Pack
      {...props}
      ref={ref}
      className={CSS.B("dropdown__container")}
      direction="y"
      reverse={location === "top"}
    >
      {children[0]}
      <Space
        className={CSS(
          CSS.bordered(),
          CSS.B("dropdown__dialog"),
          CSS.loc(location),
          CSS.visible(visible)
        )}
        empty
      >
        {children[1]}
      </Space>
    </Pack>
  )
);
Dropdown.displayName = "Dropdown";
