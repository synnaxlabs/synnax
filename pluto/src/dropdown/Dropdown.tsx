// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/dropdown/Dropdown.css";

import {
  box,
  invert,
  type location as loc,
  location,
  position,
  xy,
} from "@synnaxlabs/x";
import {
  type CSSProperties,
  type ReactElement,
  type ReactNode,
  useCallback,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Dialog as CoreDialog } from "@/dialog";
import { useClickOutside, useCombinedRefs, useResize, useSyncedRef } from "@/hooks";
import { Triggers } from "@/triggers";
import { ComponentSize } from "@/util/component";
import { findParent } from "@/util/findParent";
import { getRootElement } from "@/util/rootElement";

export type UseProps = CoreDialog.UseProps;
export type UseReturn = CoreDialog.UseReturn;

export const use = CoreDialog.use;

export type Variant = "connected" | "floating" | "modal";

/** Props for the {@link Dialog} component. */
export interface DialogProps
  extends Pick<CoreDialog.UseReturn, "visible" | "close">,
    Partial<Omit<CoreDialog.UseReturn, "visible" | "ref" | "close">>,
    Omit<Align.PackProps, "ref" | "reverse" | "size" | "empty"> {
  location?: loc.Y | loc.XY;
  children: [ReactNode, ReactNode];
  keepMounted?: boolean;
  variant?: Variant;
  maxHeight?: ComponentSize | number;
  zIndex?: number;
}

interface State {
  left: number;
  width: number;
  dialogLoc: loc.XY;
  top?: number;
  bottom?: number;
}

const ZERO_STATE: State = {
  width: 0,
  left: 0,
  dialogLoc: location.BOTTOM_LEFT,
};

const Z_INDEX_VARIABLE = "--pluto-dropdown-z-index";

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
  open,
  toggle,
  zIndex = 5,
  ...props
}: DialogProps): ReactElement => {
  const targetRef = useRef<HTMLDivElement>(null);
  const visibleRef = useSyncedRef(visible);
  const dialogRef = useRef<HTMLDivElement>(null);
  const prevLocation = useRef<location.XY | undefined>(undefined);

  const [{ dialogLoc, width, ...stateDialogStyle }, setState] = useState<State>({
    ...ZERO_STATE,
  });

  const calculatePosition = useCallback(() => {
    if (targetRef.current == null || dialogRef.current == null || !visibleRef.current)
      return;
    const f = variant === "floating" ? calcFloatingDialog : calcConnectedDialog;
    const { adjustedDialog, location } = f({
      target: targetRef.current,
      dialog: dialogRef.current,
      initial: propsLocation,
      prefer: prevLocation.current != null ? [prevLocation.current] : undefined,
    });
    prevLocation.current = location;
    const nextState: State = {
      dialogLoc: location,
      width: box.width(adjustedDialog),
      left: box.left(adjustedDialog),
    };
    if (location.y === "bottom") nextState.top = box.top(adjustedDialog);
    else {
      const windowBox = box.construct(window.document.documentElement);
      nextState.bottom = box.height(windowBox) - box.bottom(adjustedDialog);
    }
    setState(nextState);
  }, [propsLocation, variant]);

  useLayoutEffect(() => {
    calculatePosition();
  }, [visible, calculatePosition]);

  Triggers.use({ triggers: [["Escape"]], callback: close, loose: true });

  const resizeParentRef = useResize(calculatePosition, { enabled: visible });
  const combinedParentRef = useCombinedRefs(targetRef, resizeParentRef);

  const resizeDialogRef = useResize(calculatePosition, { enabled: visible });
  const combinedDialogRef = useCombinedRefs(dialogRef, resizeDialogRef);

  let dialogStyle: CSSProperties = {};
  if (variant !== "modal" && targetRef.current != null) {
    dialogStyle = { ...stateDialogStyle };
    if (variant === "connected") dialogStyle.width = width;
  }
  if (typeof maxHeight === "number") dialogStyle.maxHeight = maxHeight;
  if (visible) {
    dialogStyle.zIndex = zIndex;
    // @ts-expect-error - css variable
    dialogStyle[Z_INDEX_VARIABLE] = zIndex;
  }

  const C = variant === "connected" ? Align.Pack : Align.Space;

  useClickOutside({
    ref: dialogRef,
    exclude: (e) => {
      if (targetRef.current?.contains(e.target as Node)) return true;
      // If the target has a parent with the role of dialog, don't close the dialog.
      const parent = findParent(e.target as HTMLElement, (el) => {
        const isDialog = el?.getAttribute("role") === "dialog";
        if (!isDialog) return false;
        const zi = el.style.zIndex;
        return Number(zi) > zIndex;
      });
      return parent != null;
    },
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
  if (variant === "floating") child = createPortal(child, getRootElement());
  else if (variant === "modal") {
    child = createPortal(
      <Align.Space
        className={CSS(CSS.BE("dropdown", "bg"), CSS.visible(visible))}
        role="dialog"
        empty
        align="center"
        // @ts-expect-error - css variable
        style={{ zIndex, [Z_INDEX_VARIABLE]: zIndex }}
      >
        {child}
      </Align.Space>,
      getRootElement(),
    );
  }

  const ctxValue = useMemo(() => ({ close }), [close]);
  return (
    <CoreDialog.Context.Provider value={ctxValue}>
      <C
        {...props}
        ref={combinedParentRef}
        borderShade={4}
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
        style={{
          ...props.style,
          // @ts-expect-error - css variable
          [Z_INDEX_VARIABLE]: zIndex,
        }}
      >
        {children[0]}
        {child}
      </C>
    </CoreDialog.Context.Provider>
  );
};
Dialog.displayName = "Dropdown";

interface CalcDialogProps extends Pick<position.DialogProps, "initial" | "prefer"> {
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
  prefer,
}: CalcDialogProps): position.DialogReturn => {
  const res = position.dialog({
    container: box.construct(0, 0, window.innerWidth, window.innerHeight),
    target: box.construct(target_),
    dialog: box.construct(dialog_),
    ...FLOATING_PROPS,
    initial,
    prefer,
  });
  const { location } = res;
  const adjustedDialog = box.translate(
    res.adjustedDialog,
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
  initial = CONNECTED_PROPS.initial,
  prefer = CONNECTED_PROPS.prefer,
}: CalcDialogProps): position.DialogReturn => {
  let targetBox = box.construct(target);
  // the container is the nearest element that has a container-type or contain property

  let container = box.construct(0, 0, window.innerWidth, window.innerHeight);
  // iterate through the parent elements to find the container
  let parent = target.parentElement;
  while (parent != null) {
    const style = window.getComputedStyle(parent);
    if (style.getPropertyValue("container-type") !== "normal") {
      container = box.construct(parent);
      targetBox = box.translate(targetBox, xy.scale(box.topLeft(container), -1));
      break;
    }
    parent = parent.parentElement;
  }

  const props: position.DialogProps = {
    target: targetBox,
    dialog: box.resize(box.construct(dialog), "x", box.width(targetBox)),
    container,
    ...CONNECTED_PROPS,
    initial,
    prefer,
  };

  const res = position.dialog(props);
  const { location } = res;
  const adjustedDialog = box.translate(
    res.adjustedDialog,
    "y",
    invert(location.y === "bottom") * CONNECTED_TRANSLATE_AMOUNT,
  );

  return { adjustedDialog, location };
};
