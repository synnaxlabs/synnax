// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/dialog/Dialog.css";

import { createPortal } from "react-dom";

import { CSS } from "@/css";
import { Background } from "@/dialog/Background";
import { useContext, useInternalContext } from "@/dialog/Frame";
import { Flex } from "@/flex";
import { getRootElement } from "@/util/rootElement";

export interface DialogProps extends Flex.BoxProps<"div"> {
  passthrough?: boolean;
}

export const Dialog = ({
  style,
  background = 0,
  className,
  bordered = true,
  rounded = 1,
  passthrough = false,
  children,
  ...rest
}: DialogProps) => {
  const {
    ref,
    targetCorner,
    dialogCorner,
    style: ctxStyle,
    modalPosition,
  } = useInternalContext("Dialog.Dialog");
  const { visible, variant } = useContext();
  if (!visible && !passthrough) return null;
  const actuallyVisible =
    visible && (Object.keys(ctxStyle).length > 0 || variant === "modal");
  let dialog = (
    <Flex.Box
      pack
      ref={ref}
      y
      background={background}
      className={CSS(
        CSS.BE("dialog", "dialog"),
        CSS.loc(targetCorner.x),
        CSS.loc(targetCorner.y),
        CSS.BEM("dialog", "dialog", dialogCorner.x),
        CSS.BEM("dialog", "dialog", dialogCorner.y),
        CSS.visible(actuallyVisible),
        passthrough && CSS.BM("dialog", "passthrough"),
        CSS.M(variant),
        variant === "modal" &&
          CSS.BM("dialog", "modal", "position", modalPosition.toString()),
        className,
      )}
      rounded={rounded}
      role="dialog"
      empty
      bordered={bordered}
      align="stretch"
      style={{ ...ctxStyle, ...style }}
      {...rest}
    >
      {children}
    </Flex.Box>
  );
  if (variant === "modal")
    dialog = (
      <Background empty align="center" visible={visible}>
        {dialog}
      </Background>
    );
  if (passthrough) return dialog;
  return createPortal(dialog, getRootElement());
};
