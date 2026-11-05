// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/dialog/Dialog.css";

import { useRef } from "react";
import { createPortal } from "react-dom";

import { CSS } from "@/css";
import { Background } from "@/dialog/Background";
import { useContext, useInternalContext } from "@/dialog/Frame";
import { PORTAL_OWNER_ATTR } from "@/dialog/useClickOutside";
import { useFocusTrap } from "@/dialog/useFocusTrap";
import { Flex } from "@/flex";
import { useCombinedRefs } from "@/hooks";
import { getRootElement } from "@/util/rootElement";

/** Props for {@link Dialog}. */
export interface DialogProps extends Flex.BoxProps<"div"> {}

/**
 * The floating surface of a {@link Frame}. It mounts only while open and portals itself
 * to the document root. A modal holds the focus while open and gives it back on close.
 */
export const Dialog = ({
  style,
  background = 0,
  className,
  bordered = true,
  rounded = "small",
  children,
  ...rest
}: DialogProps) => {
  const {
    ref,
    id,
    targetCorner,
    dialogCorner,
    style: ctxStyle,
    modalPosition,
  } = useInternalContext("Dialog.Dialog");
  const { visible, variant } = useContext();
  const surfaceRef = useRef<HTMLDivElement>(null);
  const combinedRef = useCombinedRefs(ref, surfaceRef);
  const modal = variant === "modal";
  const trap = useFocusTrap(surfaceRef, modal && visible);
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
      tabIndex={modal ? -1 : undefined}
      empty
      bordered={bordered}
      align="stretch"
      style={dialogStyle}
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
