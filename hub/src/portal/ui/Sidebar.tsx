// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Icon, Select, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

export interface SidebarOrganization {
  key: string;
  name: string;
}

export interface SidebarProps {
  path: string;
  staff: boolean;
  /** teams are the enterprise organizations the user belongs to; none means a
   * personal account. */
  teams: SidebarOrganization[];
  selected: string | null;
}

interface Link {
  href: string;
  label: string;
  icon: ReactElement;
  /** owns are the path prefixes this link stays active for, beyond its own href. */
  owns?: string[];
}

const LICENSE_PAGES = ["/portal/licenses"];

const links = (enterprise: boolean): Link[] => [
  enterprise
    ? { href: "/portal", label: "Licenses", icon: <Icon.Access />, owns: LICENSE_PAGES }
    : { href: "/portal", label: "Desktop", icon: <Icon.Device />, owns: LICENSE_PAGES },
  { href: "/portal/account", label: "Account", icon: <Icon.User /> },
];

const STAFF_LINKS: Link[] = [
  { href: "/portal/staff/licenses", label: "All licenses", icon: <Icon.Policy /> },
];

const active = (path: string, link: Link): boolean =>
  path === link.href || (link.owns ?? []).some((p) => path.startsWith(p));

const withOrg = (href: string, org: string | null): string =>
  org == null ? href : `${href}?org=${org}`;

/** Sidebar is the portal's section navigation and organization switcher. */
export const Sidebar = ({
  path,
  staff,
  teams,
  selected,
}: SidebarProps): ReactElement => {
  const handleOrganization = useCallback((key: string) => {
    const url = new URL(window.location.href);
    url.searchParams.set("org", key);
    window.location.assign(url.toString());
  }, []);

  return (
    <Flex.Box y gap="large" className="portal-sidebar">
      {teams.length > 1 && (
        <Select.Static<string, SidebarOrganization>
          resourceName="Organization"
          data={teams}
          value={selected ?? teams[0].key}
          onChange={handleOrganization}
          full="x"
        />
      )}
      <Flex.Box y gap="tiny">
        {links(teams.length > 0).map((link) => (
          <NavLink key={link.href} link={link} path={path} org={selected} />
        ))}
      </Flex.Box>
      {staff && (
        <Flex.Box y gap="tiny">
          <Text.Text level="small" color={9} weight={500} style={{ padding: "0 2rem" }}>
            Staff
          </Text.Text>
          {STAFF_LINKS.map((link) => (
            <NavLink key={link.href} link={link} path={path} org={null} />
          ))}
        </Flex.Box>
      )}
    </Flex.Box>
  );
};

interface NavLinkProps {
  link: Link;
  path: string;
  org: string | null;
}

const NavLink = ({ link, path, org }: NavLinkProps): ReactElement => {
  const isActive = active(path, link);
  return (
    <Button.Button
      href={withOrg(link.href, org)}
      variant="text"
      full="x"
      gap="medium"
      textColor={isActive ? 11 : 9}
      weight={isActive ? 500 : 400}
      background={isActive ? 2 : undefined}
      className="portal-sidebar__link"
    >
      {link.icon}
      {link.label}
    </Button.Button>
  );
};
