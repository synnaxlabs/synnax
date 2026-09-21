// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  Dialog,
  Modal as PModal,
  Nav,
  Status,
  Triggers,
} from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

export interface FrameProps extends PropsWithChildren {
  /** trigger opens the modal. Leave it out to control visibility from outside. */
  trigger?: ReactNode;
  visible?: boolean;
  onVisibleChange?: Dialog.FrameProps["onVisibleChange"];
  /** name is the modal's title. Dots split it into breadcrumb segments. */
  name: string;
  icon?: PModal.HeaderProps["icon"];
  className?: string;
}

/**
 * Frame is a portal modal: the Pluto modal anatomy under the portal's density scope,
 * opened by `trigger`. Children render inside the dialog and read
 * `Dialog.useContext().close` to dismiss it.
 */
export const Frame = ({
  trigger,
  visible,
  onVisibleChange,
  name,
  icon,
  className,
  children,
}: FrameProps): ReactElement => (
  <Dialog.Frame variant="modal" visible={visible} onVisibleChange={onVisibleChange}>
    {trigger}
    <PModal.Frame className={`portal-modal ${className ?? ""}`}>
      <PModal.Header icon={icon}>{name}</PModal.Header>
      {children}
    </PModal.Frame>
  </Dialog.Frame>
);

export const Body = PModal.Body;

export interface FooterProps {
  /** error reads at the start of the footer when set. */
  error?: string | null;
  children: ReactNode;
}

/** Footer is the modal's action bar. Put the primary action last. */
export const Footer = ({ error, children }: FooterProps): ReactElement => (
  <PModal.Footer>
    {error != null && (
      <Nav.Bar.Start>
        <Status.Summary variant="error" level="small" message={error} />
      </Nav.Bar.Start>
    )}
    <Nav.Bar.End x align="center" gap="small">
      {children}
    </Nav.Bar.End>
  </PModal.Footer>
);

/** Cancel closes the enclosing modal. */
export const Cancel = ({ children = "Cancel" }: PropsWithChildren): ReactElement => {
  const { close } = Dialog.useContext();
  return (
    <Button.Button
      variant="outlined"
      onClick={close}
      triggerIndicator={Triggers.ESCAPE}
    >
      {children}
    </Button.Button>
  );
};
