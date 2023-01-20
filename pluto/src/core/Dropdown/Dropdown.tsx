// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, RefObject, useCallback, useRef, useState } from "react";

import clsx from "clsx";

import { Space, SpaceProps } from "@/core/Space";

import { Pack } from "../Pack";

import { useClickOutside } from "@/hooks";
import { YLocation } from "@/spatial";
import { locationCls, visibleCls } from "@/util/css";

import "./Dropdown.css";

export interface UseDropdownReturn {
  visible: boolean;
  close: () => void;
  open: () => void;
  toggle: (vis?: boolean) => void;
  ref: RefObject<HTMLDivElement>;
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

export interface DropdownProps
  extends Pick<UseDropdownReturn, "visible">,
    Omit<SpaceProps, "ref" | "reverse" | "size" | "empty"> {
  location?: YLocation;
  children: [JSX.Element, JSX.Element];
  keepMounted?: boolean;
}

/**
 * A controlled dropdown component that wraps its parent. For the simplest case, use
 * the {@link useDropdown} hook (more behavioral details explained there).
 *
 * @param props The props for the dropdown component. Unused props are passed to the
 * parent elment.
 *
 * @param props.visible Whether the dropdown is visible or not. This is a controlled
 *
 * @param props.children
 *
 */
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
      className="pluto-dropdown__container"
      direction="y"
      reverse={location === "top"}
    >
      {children[0]}
      <Space
        className={clsx(
          "pluto-bordered",
          "pluto-dropdown__dialog",
          locationCls(location),
          visibleCls(visible)
        )}
        empty
      >
        {children[1]}
      </Space>
    </Pack>
  )
);

Dropdown.displayName = "Dropdown";
