// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIContext } from "astro";

import { open, type Portal } from "@/portal";
import { type Organization, type OrganizationKind } from "@/server/db/schema";
import { HTTPError } from "@/server/errors";
import { organizationsFor, pick } from "@/server/organization";
import { type Session } from "@/server/session";
import { scoped } from "@/shell";

export interface Loaded {
  portal: Portal;
  session: Session;
}

/**
 * load opens the portal for a signed-in page. A signed-out visitor gets a redirect to
 * sign in that returns them here afterwards.
 */
export const load = async (context: APIContext): Promise<Loaded | Response> => {
  const portal = open(context);
  try {
    return { portal, session: await portal.session() };
  } catch (err) {
    if (!(err instanceof HTTPError) || err.status !== 401)
      throw err instanceof Error ? err : new Error(String(err));
    const back = context.url.pathname + context.url.search;
    return context.redirect(`/sign-in?redirect_url=${encodeURIComponent(back)}`, 303);
  }
};

export interface Scoped extends Loaded {
  /** organizations are every organization the user can act for, personal first. */
  organizations: Organization[];
  /** scope is the organization the page acts for, from `?org=` or the default. */
  scope: Organization;
}

/** SCOPE_COOKIE remembers the last scope chosen through `?org=`. */
const SCOPE_COOKIE = "scope";

const YEAR_SECONDS = 365 * 24 * 60 * 60;

/**
 * loadScoped opens the portal for a signed-in page that acts for one organization.
 * A page without `?org=` acts for the last scope the user chose. An `?org=` the user
 * is not a member of redirects to their default overview. A page that serves only one
 * `kind` of scope redirects any other to its overview.
 */
export const loadScoped = async (
  context: APIContext,
  kind?: OrganizationKind,
): Promise<Scoped | Response> => {
  const loaded = await load(context);
  if (loaded instanceof Response) return loaded;
  const organizations = await organizationsFor(loaded.portal.store, loaded.session);
  const requested = context.url.searchParams.get("org");
  const remembered = context.cookies.get(SCOPE_COOKIE)?.value ?? null;
  const scope = pick(organizations, requested, remembered);
  if (scope == null) return context.redirect("/", 303);
  if (requested != null)
    context.cookies.set(SCOPE_COOKIE, scope.key, {
      path: "/",
      httpOnly: true,
      sameSite: "lax",
      secure: context.url.protocol === "https:",
      maxAge: YEAR_SECONDS,
    });
  if (kind != null && scope.kind !== kind)
    return context.redirect(scoped("/", scope), 303);
  return { ...loaded, organizations, scope };
};

/** failure reads the status and message a page shows for an error it caught. */
export const failure = (err: unknown): { status: number; message: string } => {
  if (err instanceof HTTPError) return { status: err.status, message: err.message };
  console.error(err);
  return { status: 500, message: "Something went wrong. Try again." };
};
