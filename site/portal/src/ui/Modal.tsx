// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Modal as Base } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

export interface FrameProps
  extends PropsWithChildren, Pick<Dialog.FrameProps, "visible" | "onVisibleChange"> {
  /** trigger opens the modal. Leave it out to control visibility from outside. */
  trigger?: ReactNode;
  /** name is the modal's title. Dots split it into breadcrumb segments. */
  name: string;
  icon?: Base.HeaderProps["icon"];
  className?: string;
}

/**
 * Frame is a portal modal: the Lyra modal anatomy under the portal's density scope,
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
    <Base.Frame className={`portal-modal ${className ?? ""}`}>
      <Base.Header icon={icon}>{name}</Base.Header>
      {children}
    </Base.Frame>
  </Dialog.Frame>
);

export const Body = Base.Body;

export interface FooterProps {
  /** error reads at the start of the footer when set. */
  error?: string | null;
  /** hint reads at the start of the footer when there is no error, e.g. to say an
   * action needs a press and hold. */
  hint?: string;
  children: ReactNode;
}

/** Footer is the modal's action bar. Put the primary action last. */
export const Footer = ({ error, hint, children }: FooterProps): ReactElement => (
  <Base.Footer>
    {error != null ? (
      <Nav.Bar.Start>
        <Status.Summary variant="error" level="small" message={error} />
      </Nav.Bar.Start>
    ) : (
      hint != null && (
        <Nav.Bar.Start>
          <Text.Text level="small" color={9}>
            {hint}
          </Text.Text>
        </Nav.Bar.Start>
      )
    )}
    <Nav.Bar.End x align="center" gap="small">
      {children}
    </Nav.Bar.End>
  </Base.Footer>
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
