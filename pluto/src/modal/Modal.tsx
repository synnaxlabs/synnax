// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/modal/Modal.css";

import { ReactElement, useCallback, useRef } from "react";
import { createPortal } from "react-dom";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Dialog as Core } from "@/dialog";
import { useClickOutside } from "@/hooks";
import { Triggers } from "@/triggers";
import { findParent } from "@/util/findParent";
import { getRootElement } from "@/util/rootElement";

export interface ModalProps
  extends Pick<Core.UseReturn, "visible" | "close">,
    Align.SpaceProps {
  centered?: boolean;
  enabled?: boolean;
  root?: string;
}

export const Dialog = ({
  children,
  centered,
  visible,
  enabled = true,
  close,
  style,
  ...props
}: ModalProps): ReactElement => {
  const dialogRef = useRef<HTMLDivElement>(null);
  const visibleRef = useRef(visible);
  useClickOutside({
    ref: dialogRef,
    exclude: (e: MouseEvent) => {
      const parent = findParent(
        e.target as HTMLElement,
        (e) => (e as HTMLElement).getAttribute("role") === "dialog",
      );
      return parent != null;
    },
    onClickOutside: close,
  });

  const handleTrigger = useCallback(
    (e: Triggers.UseEvent) => {
      if (!visibleRef.current || e.stage !== "start") return;
      const visChildren = dialogRef.current?.querySelectorAll(`.${CSS.visible(true)}`);
      if (visChildren && visChildren.length > 0) return;
      close();
    },
    [close],
  );
  Triggers.use({ triggers: [["Escape"]], callback: handleTrigger, loose: true });
  return (
    <Align.Space
      className={CSS(
        CSS.BE("modal", "bg"),
        CSS.visible(visible),
        enabled && CSS.M("enabled-modal"),
      )}
      empty
      align="center"
    >
      <Align.Space
        className={CSS(CSS.BE("modal", "dialog"), centered && CSS.M("centered"))}
        role="dialog"
        empty
        ref={dialogRef}
        {...props}
        style={{ zIndex: enabled ? 11 : undefined, ...style }}
      >
        <Align.Space className={CSS(CSS.BE("modal", "content"))} empty>
          {children}
        </Align.Space>
      </Align.Space>
    </Align.Space>
  );
};

export const Modal = ({ root, ...props }: ModalProps): ReactElement =>
  createPortal(<Dialog {...props} />, getRootElement(root));
