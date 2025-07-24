// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/dialog/Dialog.css";

import { createPortal } from "react-dom";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Background } from "@/dialog/Background";
import { useContext, useInternalContext } from "@/dialog/Frame";
import { getRootElement } from "@/util/rootElement";

export interface DialogProps extends Align.SpaceProps {}

export const Dialog = ({
  style,
  background = 0,
  className,
  bordered,
  ...rest
}: DialogProps) => {
  const { ref, location, style: ctxStyle } = useInternalContext();
  const { visible, variant } = useContext();
  bordered ??= variant === "modal";
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
        className,
      )}
      role="dialog"
      empty
      bordered={bordered}
      style={{ ...ctxStyle, ...style }}
      {...rest}
    />
  );
  if (variant === "floating") dialog = createPortal(dialog, getRootElement());
  else if (variant === "modal")
    dialog = createPortal(
      <Background empty align="center" visible={visible}>
        {dialog}
      </Background>,
      getRootElement(),
    );
  return dialog;
};
