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
  type ReactElement,
  useCallback,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";

import { box, location as loc, xy } from "@synnaxlabs/x";
import { createPortal } from "react-dom";

import { Align } from "@/align";
import { CSS } from "@/css";
import { useClickOutside, useResize, useCombinedRefs, useSyncedRef } from "@/hooks";
import { chooseLocation } from "@/tooltip/Dialog";
import { Triggers } from "@/triggers";
import { findParent } from "@/util/findParent";

import "@/dropdown/Dropdown.css";

/** Props for the {@link use} hook. */
export interface UseProps {
  initialVisible?: boolean;
  onVisibleChange?: (vis: boolean) => void;
}

/** Return type for the {@link use} hook. */
export interface UseReturn {
  visible: boolean;
  close: () => void;
  open: () => void;
  toggle: (vis?: boolean | unknown) => void;
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
  const toggle = useCallback(
    (vis?: boolean | unknown) =>
      setVisible((v) => {
        if (typeof vis === "boolean") return vis;
        return !v;
      }),
    [setVisible, onVisibleChange],
  );
  const open = useCallback(() => toggle(true), [toggle]);
  const close = useCallback(() => toggle(false), [toggle]);
  Triggers.use({ triggers: [["Escape"]], callback: close, loose: true });
  return { visible, open, close, toggle };
};

/** Props for the {@link Dialog} component. */
export interface DialogProps
  extends Pick<UseReturn, "visible" | "close">,
    Partial<Omit<UseReturn, "visible" | "ref" | "close">>,
    Omit<Align.PackProps, "ref" | "reverse" | "size" | "empty"> {
  location?: loc.Y | loc.XY;
  children: [ReactNode, ReactNode];
  keepMounted?: boolean;
  variant?: "connected" | "floating";
}

interface State {
  pos: xy.XY;
  loc: loc.XY;
  width: number;
}

const ZERO_STATE: State = {
  pos: xy.ZERO,
  loc: { x: "left", y: "bottom" },
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
export const Dialog = ({
  visible,
  children,
  location,
  keepMounted = true,
  className,
  variant = "connected",
  close,
  // It's common to pass these in, so we'll destructure and ignore them so we don't
  // get an invalid prop on div tag error.
  open: _o,
  toggle: _t,
  ...props
}: DialogProps): ReactElement => {
  const targetRef = useRef<HTMLDivElement>(null);
  const visibleRef = useRef<boolean | null>(null);
  const dialogRef = useRef<HTMLDivElement>(null);

  const [{ pos, loc: loc_, width }, setState] = useState<State>({ ...ZERO_STATE });
  const locationRef = useSyncedRef(location);

  const calculatePosition = useCallback(() => {
    if (targetRef.current == null) return;
    const windowBox = box.construct(0, 0, window.innerWidth, window.innerHeight);
    let targetBox = box.construct(targetRef.current);
    // Look for parent elements of the box that are absolutely positioned
    const parent =
      variant === "floating"
        ? document.documentElement
        : findParent(targetRef.current, (el) => {
            if (el === null) return false;
            const style = window.getComputedStyle(el);
            return style.position === "absolute";
          });
    if (parent != null) {
      const parentBox = box.construct(parent);
      targetBox = box.translate(targetBox, xy.scale(box.topLeft(parentBox), -1));
    }
    const xyLoc = chooseLocation(locationRef.current, targetBox, windowBox);
    if (xyLoc.x === "center") xyLoc.x = "left";
    const pos = xy.construct(
      box.loc(targetBox, loc.swap(xyLoc.x)),
      box.loc(targetBox, xyLoc.y),
    );
    setState({ pos, loc: { ...xyLoc }, width: box.width(targetBox) });
  }, [variant]);

  if (targetRef.current != null) {
    if (visible && (visibleRef.current == null || !visibleRef.current)) {
      calculatePosition();
    }
    visibleRef.current = visible;
  }

  const resizeRef = useResize(calculatePosition);
  const combinedRef = useCombinedRefs(targetRef, resizeRef);
  const dialogStyle: CSSProperties = { ...xy.css(pos) };
  if (variant === "connected") dialogStyle.width = width;

  const C = variant === "connected" ? Align.Pack : Align.Space;

  useClickOutside({
    ref: dialogRef,
    exclude: [targetRef],
    onClickOutside: close,
  });

  let child: ReactElement = (
    <Align.Space
      ref={dialogRef}
      className={CSS(
        CSS.BE("dropdown", "dialog"),
        CSS.loc(loc_.x),
        CSS.loc(loc_.y),
        CSS.visible(visible),
        CSS.M(variant),
      )}
      role="dialog"
      empty
      style={dialogStyle}
    >
      {(keepMounted || visible) && children[1]}
    </Align.Space>
  );
  if (variant === "floating") child = createPortal(child, document.body);

  return (
    <C
      {...props}
      ref={combinedRef}
      className={CSS(className, CSS.B("dropdown"), CSS.visible(visible))}
      direction="y"
      reverse={loc_.y === "top"}
    >
      {children[0]}
      {child}
    </C>
  );
};
Dialog.displayName = "Dropdown";
