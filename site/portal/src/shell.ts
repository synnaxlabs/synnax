// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schema } from "@/server/db";

/** HOME is where a login lands when nothing sent the visitor. */
export const HOME = "/";

const PLACEHOLDER_ORIGIN = "http://portal.invalid";

/**
 * landing reads where a login lands from its `redirect_url`: the page that sent the
 * visitor, or {@link HOME} for anything but a same-site path.
 */
export const landing = (search: URLSearchParams): string => {
  const raw = search.get("redirect_url");
  if (raw == null) return HOME;
  // Resolving the way a browser does catches every form that escapes the origin.
  const url = URL.parse(raw, PLACEHOLDER_ORIGIN);
  if (url?.origin !== PLACEHOLDER_ORIGIN) return HOME;
  return `${url.pathname}${url.search}${url.hash}`;
};

export type Tab = "overview" | "devices" | "licenses" | "members" | "admin";

export interface Link {
  tab: Tab;
  label: string;
  href: string;
}

/** invitationPath is the page an invitation's email links to. */
export const invitationPath = (key: string): string => `/invitations/${key}`;

/** activatePath is the page that activates a machine, open on `license` if given. */
export const activatePath = (license?: string): string =>
  license == null ? "/licenses/activate" : `/licenses/activate?license=${license}`;

/** scoped adds the scope to a portal path, so a link keeps the scope it came from. */
export const scoped = (path: string, scope: schema.Organization): string =>
  `${path}?org=${scope.key}`;

/**
 * tabs returns the tab row for a scope. A personal scope has its devices, a team scope
 * its licenses and members. Staff get an admin tab in every scope.
 */
export const tabs = (scope: schema.Organization, staff: boolean): Link[] => {
  const sections: Link[] =
    scope.kind === "personal"
      ? [
          { tab: "overview", label: "Overview", href: scoped("/", scope) },
          { tab: "devices", label: "Devices", href: scoped("/devices", scope) },
        ]
      : [
          { tab: "overview", label: "Overview", href: scoped("/", scope) },
          { tab: "licenses", label: "Licenses", href: scoped("/licenses", scope) },
          { tab: "members", label: "Members", href: scoped("/members", scope) },
        ];
  if (staff) sections.push({ tab: "admin", label: "Admin", href: "/admin" });
  return sections;
};

export type AdminSection = "organizations" | "licenses" | "activity";

export interface AdminLink {
  section: AdminSection;
  label: string;
  href: string;
}

/** ADMIN_LINKS are the sections of the staff console, in the order they show. */
export const ADMIN_LINKS: AdminLink[] = [
  { section: "organizations", label: "Organizations", href: "/admin" },
  { section: "licenses", label: "Licenses", href: "/admin/licenses" },
  { section: "activity", label: "Activity", href: "/admin/activity" },
];
