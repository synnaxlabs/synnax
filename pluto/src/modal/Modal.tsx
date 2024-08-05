// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/modal/Modal.css";

import { ReactElement, useRef } from "react";
import { createPortal } from "react-dom";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { useClickOutside } from "@/hooks";
import { Triggers } from "@/triggers";
import { getRootElement } from "@/util/rootElement";

export interface ModalProps
  extends Pick<Dialog.UseReturn, "visible" | "close">,
    Align.SpaceProps {}

export const Modal = ({
  visible,
  children,
  close,
  ...props
}: ModalProps): ReactElement => {
  const ref = useRef<HTMLDivElement>(null);
  useClickOutside({ ref, onClickOutside: close });
  Triggers.use({ triggers: [["Escape"]], callback: close, loose: true });
  return createPortal(
    <Align.Space
      className={CSS(CSS.BE("modal", "bg"), CSS.visible(visible))}
      role="dialog"
      empty
      align="center"
    >
      <Align.Space
        className={CSS(CSS.BE("modal", "dialog"))}
        role="dialog"
        empty
        ref={ref}
        {...props}
      >
        <Align.Space className={CSS(CSS.BE("modal", "content"))} role="dialog" empty>
          {children}
        </Align.Space>
      </Align.Space>
    </Align.Space>,
    getRootElement(),
  );
};
