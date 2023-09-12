// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type CSSProperties,
  forwardRef,
  type ReactElement,
  type RefObject,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import { box, location as loc, xy } from "@synnaxlabs/x";

import { Align } from "@/align";
import { CSS } from "@/css";
import { useClickOutside, useResize } from "@/hooks";
import { useCombinedRefs } from "@/hooks/useCombineRefs";
import { Triggers } from "@/triggers";

import "@/dropdown/Dropdown.css";

/** Props for the {@link use} hook. */
export interface UseProps {
  initialVisible?: boolean;
  onVisibleChange?: (vis: boolean) => void;
}

/** Return type for the {@link use} hook. */
export interface UseReturn {
  visible: boolean;
  ref: RefObject<HTMLDivElement>;
  close: () => void;
  open: () => void;
  toggle: (vis?: boolean) => void;
}

/**
 * Implements basic dropdown behavior, and should be preferred when using
 * the {@link Dialog} component. Opens the dropdown whenever the 'open' function is
 * called, and closes it whenever the 'close' function is called OR the user clicks
 * outside of the dropdown parent wrapped,which includes the dropdown trigger (often
 * a button or input).
 *
 * @param initialVisible - Whether the dropdown should be visible on mount.
 * @returns visible - Whether the dropdown is visible.
 * @returns ref - The ref to the dropdown parent.
 * @returns close - A function to close the dropdown.
 * @returns open - A function to open the dropdown.
 * @returns toggle - A function to toggle the dropdown.
 */
export const use = (props?: UseProps): UseReturn => {
  const { initialVisible = false, onVisibleChange } = props ?? {};
  const [visible, setVisible] = useState(initialVisible);
  useEffect(() => onVisibleChange?.(visible), [visible, onVisibleChange]);
  const ref = useRef<HTMLDivElement>(null);
  const toggle = useCallback(
    (vis?: boolean) => setVisible((v) => vis ?? !v),
    [setVisible, onVisibleChange],
  );
  const open = useCallback(() => toggle(true), [toggle]);
  const close = useCallback(() => toggle(false), [toggle]);
  useClickOutside(ref, close);
  Triggers.use({ triggers: [["Escape"]], callback: close, loose: true });
  return { visible, ref, open, close, toggle };
};

/** Props for the {@link Dialog} component. */
export interface DialogProps
  extends Pick<UseReturn, "visible">,
    Omit<Align.PackProps, "ref" | "reverse" | "size" | "empty"> {
  location?: loc.Y;
  children: [ReactElement, ReactElement];
  keepMounted?: boolean;
  matchTriggerWidth?: boolean;
}

interface State {
  pos: xy.XY;
  loc: loc.Location;
  width: number;
}

const ZERO_STATE: State = {
  pos: xy.ZERO,
  loc: "top",
  width: 0,
};

/**
 * A controlled dropdown dialog component that wraps its children. For the simplest case, use
 * the {@link use} hook (more behavioral details explained there).
 *
 * @param props - The props for the dropdown component. Unlisted props are passed to the
 * parent elment.
 * @param props.visible - Whether the dropdown is visible or not. This is a controlled
 * @param props.children - Two children are expected: the dropdown trigger (often a button
 * or input) and the dropdown content.
 */
export const Dialog = forwardRef<HTMLDivElement, DialogProps>(
  (
    {
      visible,
      children,
      location,
      keepMounted = true,
      className,
      matchTriggerWidth = false,
      ...props
    }: DialogProps,
    forwardedRef,
  ): ReactElement => {
    const ref = useRef<HTMLDivElement>(null);
    const visibleRef = useRef<boolean | null>(null);

    const [{ pos, loc: loc_, width }, setState] = useState<State>(ZERO_STATE);

    const calculatePosition = useCallback(() => {
      if (ref.current == null) return;
      const windowBox = box.construct(0, 0, window.innerWidth, window.innerHeight);
      const b = box.construct(ref.current);
      const toTop = Math.abs(box.center(b).y - box.top(windowBox));
      const toBottom = Math.abs(box.center(b).y - box.bottom(windowBox));
      const loc_ = loc.construct(location ?? (toBottom > toTop ? "bottom" : "top"));
      const pos = xy.construct(box.left(b), box.loc(b, loc_));
      setState({ pos, loc: loc_, width: box.width(b) });
    }, []);

    if (ref.current != null) {
      if (visible && (visibleRef.current == null || !visibleRef.current)) {
        calculatePosition();
      }
      visibleRef.current = visible;
    }

    const resizeRef = useResize(calculatePosition);
    const combinedRef = useCombinedRefs(forwardedRef, ref, resizeRef);
    const dialogStyle: CSSProperties = { ...xy.css(pos) };
    if (matchTriggerWidth) dialogStyle.width = width;

    return (
      <Align.Pack
        {...props}
        ref={combinedRef}
        className={CSS(className, CSS.B("dropdown"), CSS.visible(visible))}
        direction="y"
        reverse={loc_ === "top"}
      >
        {children[0]}
        {(keepMounted || visible) && (
          <Align.Space
            className={CSS(
              CSS.BE("dropdown", "dialog"),
              CSS.loc(loc_),
              CSS.visible(visible),
            )}
            role="dialog"
            empty
            style={dialogStyle}
          >
            {children[1]}
          </Align.Space>
        )}
      </Align.Pack>
    );
  },
);
Dialog.displayName = "Dropdown";
