// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/modal/Modal.css";

import { type ReactElement, useCallback, useRef } from "react";
import { createPortal } from "react-dom";

import { Align } from "@/align";
import { CSS } from "@/css";
import { type Dialog as Core } from "@/dialog";
import { useClickOutside, useSyncedRef } from "@/hooks";
import { Triggers } from "@/triggers";
import { findParent } from "@/util/findParent";
import { getRootElement } from "@/util/rootElement";

export interface ModalProps
  extends Pick<Core.UseReturn, "visible" | "close">,
    Align.SpaceProps {
  root?: string;
  offset?: number;
}

interface BackgroundProps extends Align.SpaceProps {
  visible: boolean;
}

export const Background = ({
  children,
  visible,
  ...rest
}: BackgroundProps): ReactElement => (
  <Align.Space
    className={CSS(CSS.BE("modal", "bg"), CSS.visible(visible))}
    empty
    align="center"
    {...rest}
  >
    {children}
  </Align.Space>
);

export const Dialog = ({
  children,
  visible,
  close,
  style,
  offset = 15,
  ...rest
}: ModalProps): ReactElement => {
  const dialogRef = useRef<HTMLDivElement>(null);
  const visibleRef = useSyncedRef(visible);
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
  let dialogProps: Partial<Align.SpaceProps> = {};
  if (visible)
    dialogProps = {
      style: { zIndex: 11, ...style, top: `${offset}%` },
      rounded: 1,
      bordered: true,
      borderShade: 5,
    };

  return (
    <Background visible={visible}>
      <Align.Space role="dialog" empty ref={dialogRef} {...rest} {...dialogProps}>
        {children}
      </Align.Space>
    </Background>
  );
};

export const Modal = ({ root, ...rest }: ModalProps): ReactElement =>
  createPortal(<Dialog {...rest} />, getRootElement(root));
