// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/dialog/Dialog.css";

import {
  box,
  invert,
  type location as loc,
  location,
  position,
  xy,
} from "@synnaxlabs/x";
import {
  createContext,
  type CSSProperties,
  type PropsWithChildren,
  type ReactElement,
  type RefCallback,
  useCallback,
  useContext as reactUseContext,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

import { Align } from "@/align";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Background } from "@/dialog/Background";
import {
  useClickOutside,
  useCombinedRefs,
  useRequiredContext,
  useResize,
  useSyncedRef,
} from "@/hooks";
import { state } from "@/state";
import { Triggers } from "@/triggers";
import { findParent } from "@/util/findParent";
import { getRootElement } from "@/util/rootElement";

export type Variant = "connected" | "floating" | "modal";

/** Props for the {@link Frame} component. */
export interface FrameProps
  extends Omit<Align.PackProps, "ref" | "reverse" | "size" | "empty">,
    PropsWithChildren {
  initialVisible?: boolean;
  visible?: boolean;
  onVisibleChange?: state.Setter<boolean>;
  location?: loc.Y | loc.XY;
  variant?: Variant;
  maxHeight?: Component.Size | number;
  zIndex?: number;
  modalOffset?: number;
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

export interface ContextValue {
  close: () => void;
  open: () => void;
  toggle: () => void;
  visible: boolean;
}

const Context = createContext<ContextValue>({
  close: () => {},
  open: () => {},
  toggle: () => {},
  visible: false,
});

export const useContext = (): ContextValue => reactUseContext(Context);

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
export const Frame = ({
  children,
  location: propsLocation = "bottom",
  onPointerEnter,
  className,
  variant = "floating",
  maxHeight,
  zIndex = 5,
  modalOffset = 20,
  initialVisible = false,
  visible: propsVisible,
  onVisibleChange: propsOnVisibleChange,
  ...rest
}: FrameProps): ReactElement => {
  const [visible, setVisible] = state.usePassthrough({
    initial: initialVisible,
    value: propsVisible,
    onChange: propsOnVisibleChange,
  });
  const close = useCallback(() => setVisible(false), [setVisible]);
  const open = useCallback(() => setVisible(true), [setVisible]);
  const toggle = useCallback(() => setVisible((prev) => !prev), [setVisible]);
  const visibleRef = useSyncedRef(visible);
  const parentRef = useRef<HTMLDivElement>(null);
  const prevLocation = useRef<location.XY | undefined>(undefined);
  const dialogRef = useRef<HTMLDivElement>(null);
  const [{ dialogLoc, width, ...stateDialogStyle }, setState] = useState<State>({
    ...ZERO_STATE,
  });

  const calculatePosition = useCallback(() => {
    if (parentRef.current == null || dialogRef.current == null || !visibleRef.current)
      return;
    const calcDialog =
      variant === "floating" ? calcFloatingDialog : calcConnectedDialog;
    const { adjustedDialog, location } = calcDialog({
      target: parentRef.current,
      dialog: dialogRef.current,
      initial: propsLocation,
      prefer: prevLocation.current != null ? [prevLocation.current] : undefined,
    });
    prevLocation.current = location;
    const rounded = adjustedDialog;
    const nextState: State = {
      dialogLoc: location,
      width: box.width(rounded),
      left: box.left(rounded),
    };
    if (location.y === "bottom") nextState.top = box.top(rounded);
    else {
      const windowBox = box.construct(window.document.documentElement);
      nextState.bottom = box.height(windowBox) - box.bottom(rounded);
    }
    setState(nextState);
  }, [propsLocation, variant]);

  useLayoutEffect(() => calculatePosition(), [visible, calculatePosition]);

  Triggers.use({ triggers: [["Escape"]], callback: close, loose: true });

  let dialogStyle: CSSProperties = {};
  if (variant !== "modal" && parentRef.current != null) {
    dialogStyle = { ...stateDialogStyle };
    if (variant === "connected") dialogStyle.width = width;
  } else if (variant === "modal") dialogStyle = { top: `${modalOffset}%` };

  if (typeof maxHeight === "number") dialogStyle.maxHeight = maxHeight;
  if (visible)
    dialogStyle = {
      ...dialogStyle,
      zIndex,
      [Z_INDEX_VARIABLE]: zIndex,
    } as CSSProperties;

  const C = variant === "connected" ? Align.Pack : Align.Space;
  const resizeDialogRef = useResize(calculatePosition, { enabled: visible });
  const combinedDialogRef = useCombinedRefs(dialogRef, resizeDialogRef);

  const resizeParentRef = useResize(calculatePosition, { enabled: visible });
  const combinedParentRef = useCombinedRefs(parentRef, resizeParentRef);

  const exclude = useCallback(
    (e: { target: EventTarget | null }) => {
      if (parentRef.current?.contains(e.target as Node)) return true;
      // If the target has a parent with the role of dialog, don't close the dialog.
      const parent = findParent(e.target as HTMLElement, (el) => {
        const isDialog = el?.getAttribute("role") === "dialog";
        if (!isDialog) return false;
        const zi = el.style.zIndex;
        return Number(zi) > zIndex;
      });
      return parent != null;
    },
    [zIndex],
  );

  useClickOutside({ ref: dialogRef, exclude, onClickOutside: close });
  const internalContextValue: InternalContextValue = useMemo(
    () => ({
      ref: combinedDialogRef,
      location: dialogLoc,
      variant,
      style: dialogStyle,
    }),
    [combinedDialogRef, dialogLoc, variant, dialogStyle],
  );

  const ctxValue = useMemo(
    () => ({ close, open, toggle, visible, onPointerEnter }),
    [close, open, toggle, visible],
  );

  return (
    <Context.Provider value={ctxValue}>
      <InternalContext.Provider value={internalContextValue}>
        <C
          {...rest}
          ref={combinedParentRef}
          className={CSS(
            className,
            CSS.BE("dialog", "frame"),
            CSS.visible(visible),
            CSS.M(variant),
            CSS.loc(dialogLoc.x),
            CSS.loc(dialogLoc.y),
          )}
          y
          reverse={dialogLoc.y === "top"}
          style={{ ...rest.style, [Z_INDEX_VARIABLE]: zIndex } as CSSProperties}
        >
          {children}
        </C>
      </InternalContext.Provider>
    </Context.Provider>
  );
};
Frame.displayName = "Dropdown";

interface CalcDialogProps extends Pick<position.DialogProps, "initial" | "prefer"> {
  target: HTMLElement;
  dialog: HTMLElement;
}

const FLOATING_PROPS: Partial<position.DialogProps> = {
  alignments: ["end"],
  disable: ["center"],
  prefer: [{ y: "bottom" }],
};
const FLOATING_TRANSLATE_AMOUNT: number = 3;

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
const CONNECTED_TRANSLATE_AMOUNT: number = 0.5;

interface InternalContextValue {
  ref: RefCallback<HTMLDivElement>;
  location: loc.XY;
  variant: Variant;
  style: CSSProperties;
}

const InternalContext = createContext<InternalContextValue | null>(null);
const useInternalContext = () => useRequiredContext(InternalContext);

const calcConnectedDialog = ({
  target,
  dialog,
  initial = CONNECTED_PROPS.initial,
  prefer = CONNECTED_PROPS.prefer,
}: CalcDialogProps): position.DialogReturn => {
  const targetBox = box.construct(target);
  const win = box.construct(0, 0, window.innerWidth, window.innerHeight);
  let container = win;

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
  let adjustedDialog = box.translate(
    res.adjustedDialog,
    "y",
    invert(location.y === "bottom") * CONNECTED_TRANSLATE_AMOUNT,
  );

  const stylePropertyValueFilter = (v: string) => ["inline-size", "size"].includes(v);

  let parent: HTMLElement | null = target.parentElement;
  while (parent != null) {
    const style = window.getComputedStyle(parent);
    if (stylePropertyValueFilter(style.getPropertyValue("container-type"))) {
      container = box.construct(parent);
      if (location.y === "bottom")
        adjustedDialog = box.translate(
          adjustedDialog,
          xy.scale(box.topLeft(container), -1),
        );
      else
        adjustedDialog = box.translate(adjustedDialog, {
          x: -box.left(container),
          y: -(box.bottom(container) - box.bottom(win)),
        });
      break;
    }
    parent = parent.parentElement;
  }
  return { adjustedDialog, location };
};

export interface DialogProps extends Align.SpaceProps {
  zIndex?: number;
}

export const Dialog = ({ zIndex, style, background = 0, ...rest }: DialogProps) => {
  const { ref, location, variant, style: ctxStyle } = useInternalContext();
  const { visible } = useContext();
  if (!visible) return null;
  let dialog = (
    <Align.Pack
      ref={ref}
      y
      background={background}
      className={CSS(
        CSS.BE("dialog", "dialog"),
        CSS.loc(location.x),
        CSS.loc(location.y),
        CSS.visible(visible),
        CSS.M(variant),
      )}
      role="dialog"
      empty
      style={{ ...ctxStyle, ...style }}
      {...rest}
    />
  );
  if (variant === "floating") dialog = createPortal(dialog, getRootElement());
  else if (variant === "modal")
    dialog = createPortal(
      <Background
        role="dialog"
        empty
        align="center"
        style={{ zIndex, [Z_INDEX_VARIABLE]: zIndex } as CSSProperties}
        visible={visible}
      >
        {dialog}
      </Background>,
      getRootElement(),
    );
  return dialog;
};
