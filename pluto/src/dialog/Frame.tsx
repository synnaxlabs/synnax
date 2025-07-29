// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, location as xlocation } from "@synnaxlabs/x";
import {
  createContext,
  type CSSProperties,
  type ReactElement,
  type RefCallback,
  useCallback,
  useContext as reactUseContext,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import { Align } from "@/align";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { BACKGROUND_CLASS } from "@/dialog/Background";
import { positionDialog, type Variant } from "@/dialog/position";
import {
  useClickOutside,
  useCombinedRefs,
  useRequiredContext,
  useResize,
  useSyncedRef,
} from "@/hooks";
import { state } from "@/state";
import { Triggers } from "@/triggers";

/** Props for the {@link Frame} component. */
export interface FrameProps
  extends Omit<Align.PackProps, "ref" | "reverse" | "size" | "empty"> {
  initialVisible?: boolean;
  visible?: boolean;
  onVisibleChange?: state.Setter<boolean>;
  location?: xlocation.Y | xlocation.XY;
  variant?: Variant;
  maxHeight?: Component.Size | number;
  zIndex?: number;
  modalOffset?: number;
}

interface State {
  left: number;
  width: number;
  dialogLoc: xlocation.XY;
  top?: number;
  bottom?: number;
}

const ZERO_STATE: State = {
  width: 0,
  left: 0,
  dialogLoc: xlocation.BOTTOM_LEFT,
};

export interface ContextValue {
  close: () => void;
  open: () => void;
  toggle: () => void;
  visible: boolean;
  variant: Variant;
  location: xlocation.XY;
}

const Context = createContext<ContextValue>({
  close: () => {},
  open: () => {},
  toggle: () => {},
  variant: "floating",
  visible: false,
  location: xlocation.BOTTOM_LEFT,
});

interface InternalContextValue {
  ref: RefCallback<HTMLDivElement>;
  location: xlocation.XY;
  style: CSSProperties;
}

const InternalContext = createContext<InternalContextValue | null>(null);
export const useInternalContext = () => useRequiredContext(InternalContext);

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
  location: propsLocation,
  onPointerEnter,
  className,
  variant = "floating",
  maxHeight,
  zIndex,
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
  const close = useCallback(() => {
    setVisible(false);
  }, [setVisible]);
  const open = useCallback(() => setVisible(true), [setVisible]);
  const toggle = useCallback(() => setVisible((prev) => !prev), [setVisible]);
  const visibleRef = useSyncedRef(visible);
  const parentRef = useRef<HTMLDivElement>(null);
  const prevLocation = useRef<xlocation.XY | undefined>(undefined);
  const dialogRef = useRef<HTMLDivElement>(null);
  const [{ dialogLoc, width, ...stateDialogStyle }, setState] = useState<State>({
    ...ZERO_STATE,
  });

  const calculatePosition = useCallback(() => {
    if (parentRef.current == null || dialogRef.current == null || !visibleRef.current)
      return;
    const { adjustedDialog, location } = positionDialog({
      variant,
      target: parentRef.current,
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

  useLayoutEffect(() => calculatePosition(), [visible, calculatePosition]);

  let dialogStyle: CSSProperties = {};
  if (variant !== "modal" && parentRef.current != null) {
    dialogStyle = { ...stateDialogStyle };
    if (variant === "connected") dialogStyle.width = width;
  } else if (variant === "modal") dialogStyle = { top: `${modalOffset}%` };

  if (typeof maxHeight === "number") dialogStyle.maxHeight = maxHeight;
  if (visible) dialogStyle = { ...dialogStyle, zIndex } as CSSProperties;

  const resizeDialogRef = useResize(calculatePosition, { enabled: visible });
  const combinedDialogRef = useCombinedRefs(dialogRef, resizeDialogRef);

  const resizeParentRef = useResize(calculatePosition, { enabled: visible });
  const combinedParentRef = useCombinedRefs(parentRef, resizeParentRef);

  const exclude = useCallback(
    (e: MouseEvent) => {
      if (!visibleRef.current || dialogRef.current == null) return true;
      if (variant !== "modal") {
        const dialog = dialogRef.current;
        const visibleChildren = dialog.getElementsByClassName(CSS.visible(true));
        const exclude = visibleChildren != null && visibleChildren.length > 0;
        if (!exclude) e.stopImmediatePropagation();
        return exclude;
      }
      if (!(e.target instanceof HTMLElement)) return true;
      let dialog = e.target;
      if (dialog.className.includes(BACKGROUND_CLASS))
        dialog = dialog.children[0] as HTMLElement;
      return dialog !== dialogRef.current;
    },
    [zIndex, visibleRef],
  );

  useClickOutside({ ref: dialogRef, exclude, onClickOutside: close });
  Triggers.use({ triggers: [["Escape"]], callback: close, loose: true });

  const internalContextValue: InternalContextValue = useMemo(
    () => ({
      ref: combinedDialogRef,
      location: dialogLoc,
      style: dialogStyle,
    }),
    [combinedDialogRef, dialogLoc, dialogStyle],
  );

  const ctxValue = useMemo(
    () => ({
      close,
      open,
      toggle,
      visible,
      onPointerEnter,
      variant,
      location: dialogLoc,
    }),
    [close, open, toggle, visible, dialogLoc],
  );

  return (
    <Context.Provider value={ctxValue}>
      <InternalContext.Provider value={internalContextValue}>
        <Align.Space
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
        >
          {children}
        </Align.Space>
      </InternalContext.Provider>
    </Context.Provider>
  );
};
Frame.displayName = "Frame";
