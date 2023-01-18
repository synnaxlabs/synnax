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
import { useClickOutside } from "@/hooks";
import { YLocation } from "@/spatial";
import { visibleCls } from "@/util/css";

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
