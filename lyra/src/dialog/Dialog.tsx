// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/dialog/Dialog.css";

import { type KeyboardEvent, useRef } from "react";
import { createPortal } from "react-dom";

import { CSS } from "@/css";
import { Background } from "@/dialog/Background";
import { useContext, useInternalContext } from "@/dialog/Frame";
import { PORTAL_OWNER_ATTR } from "@/dialog/useClickOutside";
import { useDropdownFocus } from "@/dialog/useDropdownFocus";
import { useFocusTrap } from "@/dialog/useFocusTrap";
import { Flex } from "@/flex";
import { useCombinedRefs } from "@/hooks";
import { getRootElement } from "@/util/rootElement";

/** Props for {@link Dialog}. */
export interface DialogProps extends Flex.BoxProps<"div"> {}

/**
 * The floating surface of a {@link Frame}. It mounts only while open and portals itself
 * to the document root. A modal holds the focus while open and gives it back on close.
 * Any other dialog takes the place right after its trigger in the tab order.
 */
export const Dialog = ({
  ref: propsRef,
  style,
  background = 0,
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
  } = useInternalContext("Dialog.Dialog");
  const { visible, variant, close } = useContext();
  const modal = variant === "modal";
  const surfaceRef = useRef<HTMLDivElement>(null);
  const trap = useFocusTrap(surfaceRef, modal && visible);
  const dropdown = useDropdownFocus(!modal, trigger, close);
  const combinedRef = useCombinedRefs(propsRef, ref, surfaceRef, dropdown.ref);
  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>): void => {
    onKeyDown?.(e);
    dropdown.onKeyDown(e);
  };
  const dialogStyle = { ...ctxStyle, ...style };
  if (!visible) return null;
  const actuallyVisible = modal || Object.keys(ctxStyle).length > 0;
  let dialog = (
    <Flex.Box
      pack
      ref={combinedRef}
      y
      background={background}
      className={CSS.cls(
        CSS.BE("dialog", "dialog"),
        CSS.loc(targetCorner.x),
        CSS.loc(targetCorner.y),
        CSS.BEM("dialog", "dialog", dialogCorner.x),
        CSS.BEM("dialog", "dialog", dialogCorner.y),
        CSS.visible(actuallyVisible),
        CSS.M(variant),
        modal && CSS.BM("dialog", "modal", "position", modalPosition.toString()),
        className,
      )}
      rounded={rounded}
      role="dialog"
      tabIndex={-1}
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
  if (modal)
    dialog = (
      <Background empty align="center" visible={visible} {...trap}>
        {dialog}
      </Background>
    );
  return createPortal(dialog, getRootElement());
};
