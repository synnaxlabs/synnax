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
  useMemo,
  useRef,
  useState,
  useLayoutEffect,
} from "react";

import { type Component } from "@/component";
import { CSS } from "@/css";
import { BACKGROUND_CLASS } from "@/dialog/Background";
import { type LocationPreference, position } from "@/dialog/position";
import { Flex } from "@/flex";
import {
  useClickOutside,
  useCombinedRefs,
  useRequiredContext,
  useResize,
  useSyncedRef,
} from "@/hooks";
import { Menu } from "@/menu";
import { state } from "@/state";
import { Triggers } from "@/triggers";

export type Variant = "connected" | "floating" | "modal";

/** Props for the {@link Frame} component. */
export interface FrameProps
  extends Omit<Flex.BoxProps, "ref" | "reverse" | "size" | "empty"> {
  initialVisible?: boolean;
  visible?: boolean;
  onVisibleChange?: state.Setter<boolean>;
  location?: LocationPreference;
  variant?: Variant;
  maxHeight?: Component.Size | number;
  zIndex?: number;
  modalOffset?: number;
}

interface State {
  targetCorner: xlocation.XY;
  dialogCorner: xlocation.XY;
  style: CSSProperties;
}

const ZERO_STATE: State = {
  targetCorner: xlocation.BOTTOM_LEFT,
  dialogCorner: xlocation.BOTTOM_LEFT,
  style: {},
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

interface InternalContextValue
  extends Pick<State, "targetCorner" | "dialogCorner" | "style"> {
  ref: RefCallback<HTMLDivElement>;
}

const InternalContext = createContext<InternalContextValue | null>(null);
export const useInternalContext = () => useRequiredContext(InternalContext);

export const useContext = (): ContextValue => reactUseContext(Context);

const positionsEqual = (
  variant: Variant,
  next: box.Box,
  prev?: box.Box | null,
): boolean => {
  const prevNull = prev == null;
  if (prevNull) return false;
  const topLeftEqual =
    Math.abs(box.left(next) - box.left(prev)) <= 1 &&
    Math.abs(box.top(next) - box.top(prev)) <= 1;
  if (variant === "floating") return topLeftEqual;
  const widthEqual = Math.abs(box.width(next) - box.width(prev)) <= 1;
  return topLeftEqual && widthEqual;
};

const PREFERENCES: LocationPreference[] = [
  {
    targetCorner: xlocation.BOTTOM_LEFT,
    dialogCorner: xlocation.TOP_LEFT,
  },
  {
    targetCorner: xlocation.TOP_LEFT,
    dialogCorner: xlocation.BOTTOM_LEFT,
  },
  {
    targetCorner: xlocation.BOTTOM_RIGHT,
    dialogCorner: xlocation.TOP_RIGHT,
  },
  {
    targetCorner: xlocation.TOP_RIGHT,
    dialogCorner: xlocation.BOTTOM_RIGHT,
  },
  {
    targetCorner: xlocation.TOP_RIGHT,
    dialogCorner: xlocation.TOP_LEFT,
  },
];

/**
 * A controlled dropdown dialog component that wraps its children. For the simplest
 * case, use the {@link use} hook (more behavioral details explained there).
 *
 * @param props - The props for the dropdown component. Unlisted props are passed to the
 * parent element.
 * @param props.visible - Whether the dropdown is visible or not. This is a controlled
 * @param props.children - Two children are expected: the dropdown trigger (often a
 * button or input) and the dropdown content.
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
  const close = useCallback(() => setVisible(false), [setVisible]);
  const open = useCallback(() => setVisible(true), [setVisible]);
  const toggle = useCallback(() => setVisible((prev) => !prev), [setVisible]);

  const visibleRef = useSyncedRef(visible);
  const targetRef = useRef<HTMLDivElement>(null);
  const prevLocation = useRef<xlocation.XY | undefined>(undefined);
  const prevBox = useRef<box.Box | undefined>(undefined);
  const dialogRef = useRef<HTMLDivElement>(null);

  const [{ targetCorner, dialogCorner, style }, setState] = useState<State>({
    ...ZERO_STATE,
  });

  const calculatePosition = useCallback(() => {
    if (targetRef.current == null || dialogRef.current == null || !visibleRef.current)
      return;
    const target = box.construct(targetRef.current);
    let dialog = box.construct(dialogRef.current);
    if (variant === "connected") dialog = box.resize(dialog, "x", box.width(target));

    const container = box.construct(0, 0, window.innerWidth, window.innerHeight);
    const { adjustedDialog, targetCorner, dialogCorner } = position({
      target,
      dialog,
      container,
      prefer: PREFERENCES,
      initial: propsLocation,
      offset: 3,
    });
    prevLocation.current = targetCorner;
    const roundedDialog = box.round(adjustedDialog);
    if (positionsEqual(variant, roundedDialog, prevBox.current)) return;
    prevBox.current = roundedDialog;
    const style: CSSProperties = {};
    if (variant !== "modal" && targetRef.current != null) {
      style.left = box.left(roundedDialog);
      if (targetCorner.y === "top" && dialogCorner.x === targetCorner.x)
        style.bottom = box.height(container) - box.bottom(roundedDialog);
      else style.top = box.top(roundedDialog);

      if (variant === "connected") style.width = box.width(roundedDialog);
    } else if (variant === "modal") style.top = `${modalOffset}%`;
    if (typeof maxHeight === "number") style.maxHeight = maxHeight;
    if (visible) style.zIndex = zIndex;
    console.log(style);
    setState({ targetCorner: targetCorner, dialogCorner: dialogCorner, style });
  }, [propsLocation, variant]);

  const resizeDialogRef = useResize(calculatePosition, { enabled: visible });
  const combinedDialogRef = useCombinedRefs(dialogRef, resizeDialogRef);

  const resizeTargetRef = useResize(calculatePosition, { enabled: visible });
  const combinedTargetRef = useCombinedRefs(targetRef, resizeTargetRef);

  const exclude = useCallback(
    (e: MouseEvent) => {
      if (!visibleRef.current || dialogRef.current == null || targetRef.current == null)
        return true;
      if (variant !== "modal") {
        const dialog = dialogRef.current;
        const visibleChildren = dialog.getElementsByClassName(CSS.visible(true));
        let exclude = visibleChildren != null && visibleChildren.length > 0;
        if (!exclude) {
          const isTrigger = targetRef.current.contains(e.target as Node);
          exclude = isTrigger;
        }
        const contextMenus = document.getElementsByClassName(Menu.CONTEXT_MENU_CLASS);
        if (contextMenus.length > 0) exclude = true;
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
      targetCorner,
      dialogCorner,
      style,
    }),
    [combinedDialogRef, targetCorner, dialogCorner, style],
  );

  const ctxValue = useMemo(
    () => ({
      close,
      open,
      toggle,
      visible,
      onPointerEnter,
      variant,
      location: targetCorner,
    }),
    [close, open, toggle, visible, targetCorner],
  );

  return (
    <Context.Provider value={ctxValue}>
      <InternalContext.Provider value={internalContextValue}>
        <Flex.Box
          {...rest}
          ref={combinedTargetRef}
          className={CSS(
            className,
            CSS.BE("dialog", "frame"),
            CSS.visible(visible),
            CSS.M(variant),
            CSS.loc(targetCorner.x),
            CSS.loc(targetCorner.y),
            CSS.BEM("dialog", "dialog", dialogCorner.x),
            CSS.BEM("dialog", "dialog", dialogCorner.y),
          )}
          y
          reverse={targetCorner.y === "top"}
        >
          {children}
        </Flex.Box>
      </InternalContext.Provider>
    </Context.Provider>
  );
};
Frame.displayName = "Dialog.Frame";
