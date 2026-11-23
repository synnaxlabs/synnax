// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/dialog/Dialog.css";

import {
  type KeyboardEvent,
  type PropsWithChildren,
  type ReactElement,
  type SyntheticEvent,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

import { CSS } from "@/css";
import { SURFACE_CLASS, useContext, useInternalContext } from "@/dialog/Frame";
import { trapTab } from "@/dialog/tabOrder";
import { PORTAL_OWNER_ATTR } from "@/dialog/useClickOutside";
import { useDropdownFocus } from "@/dialog/useDropdownFocus";
import { Flex } from "@/flex";
import { useCombinedRefs, usePopover } from "@/hooks";
import { Portal } from "@/portal";
import { getRootElement } from "@/util";

/** The class of the native modal dialog around a modal surface. */
const BACKGROUND_CLASS = CSS.BE("dialog", "bg");

/** Whether the nearest dialog around the element is a modal. */
export const inModal = (el: Element): boolean =>
  el.closest(`.${SURFACE_CLASS}`)?.classList.contains(CSS.M("modal")) === true;

const notifyModalChange = (): void => {
  document.dispatchEvent(new Event(Portal.MODAL_CHANGE_EVENT));
};

interface ModalProps extends PropsWithChildren {
  /** The id of the frame, so a click or focus in the modal counts as inside it. */
  owner: string;
  /** Called on a `cancel` event of this dialog, whose default is always prevented. */
  onCancel: () => void;
  /** Called when the browser closes this dialog. */
  onClose: () => void;
  onKeyDown: (e: KeyboardEvent<HTMLDialogElement>) => void;
}

/**
 * The native modal dialog. Overlays in the modal portal into it. Its children mount
 * only after `showModal()`, so the browser keeps the opener to give the focus back to,
 * and a field with autoFocus can take the focus.
 */
const Modal = ({
  children,
  owner,
  onCancel,
  onClose,
  ...rest
}: ModalProps): ReactElement => {
  const ref = useRef<HTMLDialogElement>(null);
  const [shown, setShown] = useState<HTMLDialogElement | null>(null);
  const [nativeCloses, setNativeCloses] = useState(0);
  // React runs this cleanup before it removes the dialog, so close() gives the focus
  // back to the opener.
  useLayoutEffect(() => {
    const dialog = ref.current;
    if (dialog == null) return;
    // Triggers handles an Escape from inside the modal that reaches the window. The
    // browser must not also send cancel, since Chromium closes the dialog on a
    // repeated cancel.
    const handleKeyDown = (e: globalThis.KeyboardEvent): void => {
      if (e.key === "Escape" && e.target instanceof Node && dialog.contains(e.target))
        e.preventDefault();
    };
    window.addEventListener("keydown", handleKeyDown);
    dialog.showModal();
    notifyModalChange();
    setShown(dialog);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      dialog.close();
      notifyModalChange();
    };
  }, []);
  // The owner kept the frame open after the browser closed the dialog.
  useLayoutEffect(() => {
    const dialog = ref.current;
    if (nativeCloses === 0 || dialog == null || dialog.open) return;
    dialog.showModal();
    notifyModalChange();
  }, [nativeCloses]);
  // React passes cancel and close up through a nested modal, so each modal acts only
  // on its own.
  const handleCancel = (e: SyntheticEvent<HTMLDialogElement>): void => {
    e.preventDefault();
    if (e.target === e.currentTarget) onCancel();
  };
  const handleClose = (e: SyntheticEvent<HTMLDialogElement>): void => {
    if (e.target !== e.currentTarget) return;
    notifyModalChange();
    setNativeCloses((n) => n + 1);
    onClose();
  };
  return (
    <Flex.Box<"dialog">
      el="dialog"
      ref={ref}
      className={BACKGROUND_CLASS}
      empty
      align="center"
      tabIndex={-1}
      onCancel={handleCancel}
      onClose={handleClose}
      {...rest}
      {...{ [PORTAL_OWNER_ATTR]: owner }}
    >
      {shown != null && <Portal.Target element={shown}>{children}</Portal.Target>}
    </Flex.Box>
  );
};

/** Props for {@link Dialog}. */
export interface DialogProps extends Flex.BoxProps<"div"> {}

/**
 * The floating surface of a {@link Frame}. It mounts only while open and portals itself
 * into the nearest `Portal.Target`, or the document root outside one. A modal renders
 * in a native modal `<dialog>`, and the overlays in the modal portal into it. Any other
 * dialog shows in the top layer and takes the place right after its trigger in the tab
 * order.
 */
export const Dialog = ({
  ref: propsRef,
  style,
  className,
  bordered = true,
  rounded = "small",
  children,
  onKeyDown,
  ...rest
}: DialogProps) => {
  const {
    ref,
    id,
    targetCorner,
    dialogCorner,
    style: ctxStyle,
    modalPosition,
    trigger,
    cancel,
  } = useInternalContext("Dialog.Dialog");
  const { visible, placed, variant, close } = useContext();
  const modal = variant === "modal";
  const surfaceRef = useRef<HTMLDivElement>(null);
  const portalTarget = Portal.useTarget();
  const dropdown = useDropdownFocus(!modal, trigger, close);
  const combinedRef = useCombinedRefs(propsRef, ref, surfaceRef, dropdown.ref);
  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>): void => {
    onKeyDown?.(e);
    dropdown.onKeyDown(e);
  };
  const handleModalKeyDown = (e: KeyboardEvent<HTMLDialogElement>): void => {
    if (surfaceRef.current != null) trapTab(e, surfaceRef.current);
  };
  usePopover(surfaceRef, visible && !modal, portalTarget);
  const dialogStyle = { ...ctxStyle, ...style };
  if (!visible) return null;
  const dialog = (
    <Flex.Box
      pack
      ref={combinedRef}
      y
      className={CSS.cls(
        SURFACE_CLASS,
        CSS.loc(targetCorner.x),
        CSS.loc(targetCorner.y),
        CSS.BEM("dialog", "dialog", dialogCorner.x),
        CSS.BEM("dialog", "dialog", dialogCorner.y),
        CSS.visible(placed),
        CSS.M(variant),
        modal && CSS.BM("dialog", "modal", "position", modalPosition.toString()),
        className,
      )}
      rounded={rounded}
      role={modal ? undefined : "dialog"}
      popover={modal ? undefined : "manual"}
      tabIndex={modal ? undefined : -1}
      empty
      bordered={bordered}
      align="stretch"
      style={dialogStyle}
      onKeyDown={handleKeyDown}
      {...rest}
      {...{ [PORTAL_OWNER_ATTR]: id }}
    >
      {children}
    </Flex.Box>
  );
  const target = portalTarget ?? getRootElement();
  if (!modal) return createPortal(dialog, target);
  return createPortal(
    <Modal onCancel={cancel} onClose={close} onKeyDown={handleModalKeyDown} owner={id}>
      {dialog}
    </Modal>,
    target,
  );
};
