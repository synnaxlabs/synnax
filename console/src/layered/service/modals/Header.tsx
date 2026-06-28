// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Breadcrumb, Button, Icon, Nav } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useDismiss } from "@/layered/service/modals/Dismiss";

export interface HeaderProps extends Omit<Nav.BarProps, "location" | "size"> {
  /** The dotted display name; each segment renders as a breadcrumb crumb. */
  name: string;
  /** The leading icon, resolved from the icon registry when a string. */
  icon?: Icon.ReactElement | string;
}

/**
 * Header renders a modal's top title bar: a breadcrumb of the dotted name, an optional
 * leading icon, and a close button wired to dismiss the modal. Modal renderers render
 * their own Header so that title and icon stay static presentation owned by the renderer
 * rather than dynamic state on the open-modal entry.
 */
export const Header = ({ name, icon, ...rest }: HeaderProps): ReactElement => {
  const dismiss = useDismiss();
  return (
    <Nav.Bar location="top" size="6rem" bordered {...rest}>
      <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
        <Breadcrumb.Breadcrumb gap="tiny">
          {icon != null && (
            <Breadcrumb.Segment color={9}>{Icon.resolve(icon)}</Breadcrumb.Segment>
          )}
          {name.split(".").map((segment) => (
            <Breadcrumb.Segment color={9} key={segment} weight={400}>
              {segment}
            </Breadcrumb.Segment>
          ))}
        </Breadcrumb.Breadcrumb>
      </Nav.Bar.Start>
      <Nav.Bar.End style={{ paddingRight: "1rem" }}>
        <Button.Button onClick={dismiss} size="small" variant="text" textColor={9}>
          <Icon.Close />
        </Button.Button>
      </Nav.Bar.End>
    </Nav.Bar>
  );
};
