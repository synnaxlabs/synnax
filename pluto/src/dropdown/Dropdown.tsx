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
  useLayoutEffect,
} from "react";

import {
  box,
  type location as loc,
  xy,
  position,
  location,
  invert,
} from "@synnaxlabs/x";
import { createPortal } from "react-dom";

import { Align } from "@/align";
import { CSS } from "@/css";
import { useClickOutside, useResize, useCombinedRefs, useSyncedRef } from "@/hooks";
import { Triggers } from "@/triggers";

import "@/dropdown/Dropdown.css";
import { ComponentSize } from "@/util/component";

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
  variant?: "connected" | "floating" | "modal";
  maxHeight?: ComponentSize | number;
}

interface State {
  dialogBox: box.Box;
  dialogLoc: loc.XY;
}

const ZERO_STATE: State = {
  dialogBox: box.ZERO,
  dialogLoc: location.BOTTOM_LEFT,
};

/**
 * A controlled dropdown dialog component that wraps its children. For the simplest case, use
 * the {@link use} hook (more behavioral details explained there).
 *
 * @param props - The props for the dropdown component. Unlisted props are passed to the
 * parent element.
 * @param props.visible - Whether the dropdown is visible or not. This is a controlled
 * @param props.children - Two children are expected: the dropdown trigger (often a button
 * or input) and the dropdown content.
 */
export const Dialog = ({
  visible,
  children,
  location: propsLocation,
  keepMounted = true,
  className,
  variant = "connected",
  close,
  maxHeight,
  // It's common to pass these in, so we'll destructure and ignore them so we don't
  // get an invalid prop on div tag error.
  open: _o,
  toggle: _t,
  ...props
}: DialogProps): ReactElement => {
  const targetRef = useRef<HTMLDivElement>(null);
  const visibleRef = useSyncedRef(visible);
  const dialogRef = useRef<HTMLDivElement>(null);

  const [{ dialogBox, dialogLoc }, setState] = useState<State>({ ...ZERO_STATE });

  const calculatePosition = useCallback(() => {
    if (targetRef.current == null || dialogRef.current == null || !visibleRef.current)
      return;
    const f = variant === "floating" ? calcFloatingDialog : calcConnectedDialog;
    const { adjustedDialog, location } = f({
      target: targetRef.current,
      dialog: dialogRef.current,
      initial: propsLocation,
    });
    setState({ dialogLoc: location, dialogBox: adjustedDialog });
  }, [propsLocation, variant]);

  useLayoutEffect(() => {
    calculatePosition();
  }, [visible, calculatePosition]);

  const resizeParentRef = useResize(calculatePosition, { enabled: visible });
  const combinedParentRef = useCombinedRefs(targetRef, resizeParentRef);

  const resizeDialogRef = useResize(calculatePosition, { enabled: visible });
  const combinedDialogRef = useCombinedRefs(dialogRef, resizeDialogRef);

  let dialogStyle: CSSProperties = {};
  if (variant !== "modal") {
    dialogStyle = { ...xy.css(box.topLeft(dialogBox)) };
    if (variant === "connected") dialogStyle.width = box.width(dialogBox);
  }
  if (typeof maxHeight === "number") {
    dialogStyle.maxHeight = maxHeight;
  }

  const C = variant === "connected" ? Align.Pack : Align.Space;

  useClickOutside({
    ref: dialogRef,
    exclude: [targetRef],
    onClickOutside: close,
  });

  let child: ReactElement = (
    <Align.Space
      ref={combinedDialogRef}
      className={CSS(
        CSS.BE("dropdown", "dialog"),
        CSS.loc(dialogLoc.x),
        CSS.loc(dialogLoc.y),
        CSS.visible(visible),
        CSS.M(variant),
        typeof maxHeight === "string" && CSS.B(`height-${maxHeight}`),
      )}
      role="dialog"
      empty
      style={dialogStyle}
    >
      {(keepMounted || visible) && children[1]}
    </Align.Space>
  );
  if (variant === "floating") child = createPortal(child, document.body);
  else if (variant === "modal") {
    child = createPortal(
      <Align.Space
        className={CSS(CSS.BE("dropdown", "bg"), CSS.visible(visible))}
        role="dialog"
        empty
        align="center"
      >
        {child}
      </Align.Space>,
      document.body,
    );
  }

  return (
    <C
      {...props}
      ref={combinedParentRef}
      className={CSS(
        className,
        CSS.B("dropdown"),
        CSS.visible(visible),
        CSS.M(variant),
        CSS.loc(dialogLoc.x),
        CSS.loc(dialogLoc.y),
      )}
      direction="y"
      reverse={dialogLoc.y === "top"}
    >
      {children[0]}
      {child}
    </C>
  );
};
Dialog.displayName = "Dropdown";

interface CalcDialogProps extends Pick<position.DialogProps, "initial"> {
  target: HTMLElement;
  dialog: HTMLElement;
}

const FLOATING_PROPS: Partial<position.DialogProps> = {
  alignments: ["end"],
  disable: ["center"],
  prefer: [{ y: "bottom" }],
};
const FLOATING_TRANSLATE_AMOUNT: number = 6;

const calcFloatingDialog = ({
  target: target_,
  dialog: dialog_,
  initial,
}: CalcDialogProps): position.DialogReturn => {
  let { adjustedDialog, location } = position.dialog({
    container: box.construct(0, 0, window.innerWidth, window.innerHeight),
    target: box.construct(target_),
    dialog: box.construct(dialog_),
    ...FLOATING_PROPS,
    initial: initial,
  });
  adjustedDialog = box.translate(
    adjustedDialog,
    "y",
    invert(location.y === "top") * FLOATING_TRANSLATE_AMOUNT,
  );
  return { adjustedDialog, location };
};

const CONNECTED_PROPS: Partial<position.DialogProps> = {
  alignments: ["center"],
  disable: [{ y: "center" }],
  initial: { x: "center" },
  prefer: [{ y: "bottom" }],
};
const CONNECTED_TRANSLATE_AMOUNT: number = 1;

const calcConnectedDialog = ({
  target,
  dialog,
  initial,
}: CalcDialogProps): position.DialogReturn => {
  const targetBox = box.construct(target);
  const props: position.DialogProps = {
    target: targetBox,
    dialog: box.resize(box.construct(dialog), "x", box.width(targetBox)),
    container: box.construct(0, 0, window.innerWidth, window.innerHeight),
    ...CONNECTED_PROPS,
    initial: initial ?? CONNECTED_PROPS.initial,
  };

  let { adjustedDialog, location } = position.dialog(props);
  adjustedDialog = box.translate(
    adjustedDialog,
    "y",
    invert(location.y === "bottom") * CONNECTED_TRANSLATE_AMOUNT,
  );

  return { adjustedDialog, location };
};
