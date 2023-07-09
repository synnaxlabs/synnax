// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  forwardRef,
  ReactElement,
  RefObject,
  useCallback,
  useRef,
  useState,
} from "react";

import { Box, CrudeYLocation, Location } from "@synnaxlabs/x";

import { CSS } from "@/core/css";
import { useClickOutside, useResize } from "@/core/hooks";
import { useCombinedRefs } from "@/core/hooks/useCombineRefs";
import { Pack, PackProps } from "@/core/std/Pack";
import { Space } from "@/core/std/Space";
import { Triggers } from "@/core/triggers";

import "@/core/std/Dropdown/Dropdown.css";

/** Return type for the {@link useDropdown} hook. */
export interface UseDropdownReturn {
  visible: boolean;
  ref: RefObject<HTMLDivElement>;
  close: () => void;
  open: () => void;
  toggle: (vis?: boolean) => void;
}

const capitalize = (str: string): string => str[0].toUpperCase() + str.slice(1);

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
  Triggers.use({ triggers: [["Escape"]], callback: close, loose: true });
  return { visible, ref, open, close, toggle };
};

/** Props for the {@link Dropdown} component. */
export interface DropdownProps
  extends Pick<UseDropdownReturn, "visible">,
    Omit<PackProps, "ref" | "reverse" | "size" | "empty"> {
  location?: CrudeYLocation;
  children: [ReactElement, ReactElement];
  keepMounted?: boolean;
  matchTriggerWidth?: boolean;
}

// const USE_RESIZE_OPTS: UseResizeOpts = { triggers: ["moveY"] };

export const Dropdown = forwardRef<HTMLDivElement, DropdownProps>(
  (
    {
      visible,
      children,
      location,
      keepMounted = true,
      className,
      matchTriggerWidth = false,
      ...props
    }: DropdownProps,
    ref
  ): ReactElement => {
    const [[loc, dim, width], setAutoStyle] = useState<
      [CrudeYLocation, number, number]
    >([location ?? "bottom", 0, 0]);
    const handleResize = useCallback(
      (box: Box) => {
        const windowBox = new Box(0, 0, window.innerWidth, window.innerHeight);
        const distanceToTop = Math.abs(box.center.y - windowBox.top);
        const distanceToBottom = Math.abs(box.center.x - windowBox.bottom);
        const height = box.height;
        const loc = location ?? (distanceToBottom > distanceToTop ? "bottom" : "top");
        setAutoStyle([loc, height, box.width]);
      },
      [location]
    );
    const resizeRef = useResize(handleResize);
    const combinedRef = useCombinedRefs(ref, resizeRef);

    const dialogStyle = {
      [`margin${capitalize(new Location(loc).inverse.crude)}`]: dim - 1,
    };
    if (matchTriggerWidth) {
      dialogStyle.width = width;
    }

    return (
      <Pack
        {...props}
        ref={combinedRef}
        className={CSS(className, CSS.B("dropdown"), CSS.visible(visible))}
        direction="y"
        reverse={loc === "top"}
      >
        {children[0]}
        <Space
          className={CSS(CSS.BE("dropdown", "dialog"), CSS.loc(loc))}
          role="dialog"
          empty
          style={dialogStyle}
        >
          {children[1]}
        </Space>
      </Pack>
    );
  }
);
Dropdown.displayName = "Dropdown";
