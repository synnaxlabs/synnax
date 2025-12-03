// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, location } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type ReactElement,
  type RefCallback,
  useCallback,
  useMemo,
  useRef,
  useState,
} from "react";

import { type Component } from "@/component";
import { context } from "@/context";
import { CSS_CLASS as CONTEXT_MENU_CSS_CLASS } from "@/context-menu/types";
import { CSS } from "@/css";
import { BACKGROUND_CLASS } from "@/dialog/Background";
import { type LocationPreference, position, type Preference } from "@/dialog/position";
import { Flex } from "@/flex";
import {
  useClickOutside,
  useCombinedRefs,
  useResize,
  useSyncedRef,
  useWindowResize,
} from "@/hooks";
import { state } from "@/state";
import { Triggers } from "@/triggers";

export type Variant = "connected" | "floating" | "modal";

export type ModalPosition = "slammed" | "shifted" | "base";

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
  modalPosition?: ModalPosition;
}

interface State {
  targetCorner: location.XY;
  dialogCorner: location.XY;
  modalPosition: ModalPosition;
  style: CSSProperties;
}

const ZERO_STATE: State = {
  targetCorner: location.BOTTOM_LEFT,
  dialogCorner: location.BOTTOM_LEFT,
  style: {},
  modalPosition: "base",
};

export interface ContextValue {
  close: () => void;
  open: () => void;
  toggle: () => void;
  visible: boolean;
  variant: Variant;
  location: location.XY;
}

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: {
    close: () => {},
    location: location.BOTTOM_LEFT,
    open: () => {},
    toggle: () => {},
    variant: "floating",
    visible: false,
  },
  displayName: "Dialog.Context",
});
export { useContext };

interface InternalContextValue
  extends Pick<State, "targetCorner" | "dialogCorner" | "style" | "modalPosition"> {
  ref: RefCallback<HTMLDivElement>;
}

const [InternalContext, useInternalContext] = context.create<InternalContextValue>({
  displayName: "Dialog.InternalContext",
  providerName: "Dialog.Frame",
});

export { useInternalContext };

const ESCAPE_TRIGGERS: Triggers.Trigger[] = [["Escape"]];

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
    targetCorner: location.BOTTOM_LEFT,
    dialogCorner: location.TOP_LEFT,
  },
  {
    targetCorner: location.TOP_LEFT,
    dialogCorner: location.BOTTOM_LEFT,
  },
  {
    targetCorner: location.BOTTOM_RIGHT,
    dialogCorner: location.TOP_RIGHT,
  },
  {
    targetCorner: location.TOP_RIGHT,
    dialogCorner: location.BOTTOM_RIGHT,
  },
  {
    targetCorner: location.TOP_RIGHT,
    dialogCorner: location.TOP_LEFT,
  },
];

const selectModalPosition = (
  dialogHeight: number,
  windowHeight: number,
): ModalPosition => {
  if (dialogHeight / windowHeight > 0.8) return "slammed";
  if (dialogHeight / windowHeight > 0.6) return "shifted";
  return "base";
};

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
  initialVisible = false,
  visible: propsVisible,
  onVisibleChange: propsOnVisibleChange,
  modalPosition: propsModalPosition = "base",
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
  const prevLocationPreference = useRef<Preference | undefined>(undefined);
  const prevBox = useRef<box.Box | undefined>(undefined);
  const dialogRef = useRef<HTMLDivElement>(null);

  const [{ targetCorner, dialogCorner, style, modalPosition }, setState] =
    useState<State>({ ...ZERO_STATE, modalPosition: propsModalPosition });

  const calculatePosition = useCallback(() => {
    if (targetRef.current == null || dialogRef.current == null || !visibleRef.current)
      return;
    const target = box.construct(targetRef.current);
    if (box.areaIsZero(target) && variant !== "modal") return setVisible(false);

    let dialog = box.construct(dialogRef.current);
    if (variant === "connected") dialog = box.resize(dialog, "x", box.width(target));
    const windowBox = box.construct(0, 0, window.innerWidth, window.innerHeight);
    if (variant === "modal") {
      const modalPosition = selectModalPosition(
        box.height(dialog),
        box.height(windowBox),
      );
      return setState((prev) => {
        if (modalPosition === prev.modalPosition) return prev;
        return { ...prev, modalPosition };
      });
    }
    let prefer = PREFERENCES;
    if (prevLocationPreference.current != null)
      prefer = [prevLocationPreference.current, ...PREFERENCES];
    // In the connected or floating case, we use a more sophisticated positioning
    // algorithm.
    const { adjustedDialog, ...locations } = position({
      target,
      dialog,
      container: windowBox,
      prefer,
      initial: propsLocation,
      offset: 3,
    });
    prevLocationPreference.current = locations;
    const { targetCorner, dialogCorner } = locations;
    const roundedDialog = box.round(adjustedDialog);
    if (positionsEqual(variant, roundedDialog, prevBox.current)) return;
    prevBox.current = roundedDialog;
    const style: CSSProperties = {};
    style.left = box.left(roundedDialog);
    if (targetCorner.y === "top" && dialogCorner.x === targetCorner.x)
      style.bottom = box.height(windowBox) - box.bottom(roundedDialog);
    else style.top = box.top(roundedDialog);
    if (variant === "connected") style.width = box.width(roundedDialog);
    if (typeof maxHeight === "number") style.maxHeight = maxHeight;
    if (visible) style.zIndex = zIndex;
    setState((prev) => ({ ...prev, targetCorner, dialogCorner, style }));
  }, [propsLocation, variant]);

  const resizeDialogRef = useResize(calculatePosition, { enabled: visible });
  const combinedDialogRef = useCombinedRefs(dialogRef, resizeDialogRef);

  const resizeTargetRef = useResize(calculatePosition, { enabled: visible });
  const combinedTargetRef = useCombinedRefs(targetRef, resizeTargetRef);

  useWindowResize(calculatePosition);

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
        const contextMenus = document.getElementsByClassName(CONTEXT_MENU_CSS_CLASS);
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

  const handleEscape = useCallback(
    ({ stage }: Triggers.UseEvent) => {
      if (
        stage !== "start" ||
        visibleRef.current === false ||
        dialogRef.current == null
      )
        return;
      if (variant !== "modal") return close();
      const dialogEl = dialogRef.current;
      const allModals = Array.from(document.getElementsByClassName(BACKGROUND_CLASS));
      const thisModalIndex = allModals.findIndex((e) => e.contains(dialogEl));
      const children = dialogEl.getElementsByClassName(CSS.visible(true));
      if (thisModalIndex === allModals.length - 1 && children.length === 0)
        return close();
    },
    [close, variant],
  );

  useClickOutside({ ref: dialogRef, exclude, onClickOutside: close });
  Triggers.use({ triggers: ESCAPE_TRIGGERS, callback: handleEscape, loose: true });

  const internalContextValue: InternalContextValue = useMemo(
    () => ({
      ref: combinedDialogRef,
      targetCorner,
      dialogCorner,
      style,
      modalPosition,
    }),
    [combinedDialogRef, targetCorner, dialogCorner, style, modalPosition],
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
    <Context value={ctxValue}>
      <InternalContext value={internalContextValue}>
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
      </InternalContext>
    </Context>
  );
};
Frame.displayName = "Dialog.Frame";
