// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/modals/Header.css";

import { Breadcrumb, Button, Dialog, Icon, Nav } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface HeaderProps extends Omit<
  Nav.BarProps,
  "location" | "size" | "children"
> {
  /** Dotted breadcrumb name, or pre-split segments for names that may contain dots. */
  children: string | string[];
  icon?: Icon.ReactElement;
  hideClose?: boolean;
}

/**
 * Header renders a modal's top title bar: a breadcrumb of the given name, an optional
 * leading icon, and a close button wired to dismiss the modal. Modal renderers render
 * their own Header so that title and icon stay static presentation owned by the
 * renderer rather than dynamic state on the open-modal entry.
 */
export const Header = ({
  icon,
  children,
  className,
  hideClose = false,
  ...rest
}: HeaderProps): ReactElement => {
  const { close } = Dialog.useContext();
  const segments = typeof children === "string" ? children.split(".") : children;
  return (
    <Nav.Bar
      location="top"
      size="6rem"
      bordered
      className={CSS.cls(CSS.BE("modal", "header"), className)}
      {...rest}
    >
      <Nav.Bar.Start>
        <Breadcrumb.Breadcrumb gap="tiny">
          {icon != null && <Breadcrumb.Segment color={9}>{icon}</Breadcrumb.Segment>}
          {segments.map((segment) => (
            <Breadcrumb.Segment color={9} key={segment} weight={400}>
              {segment}
            </Breadcrumb.Segment>
          ))}
        </Breadcrumb.Breadcrumb>
      </Nav.Bar.Start>
      {!hideClose && (
        <Nav.Bar.End>
          <Button.Button
            aria-label="Close"
            onClick={close}
            size="small"
            variant="text"
            textColor={9}
          >
            <Icon.Close />
          </Button.Button>
        </Nav.Bar.End>
      )}
    </Nav.Bar>
  );
};
