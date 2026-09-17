// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIContext } from "astro";

import { HTTPError, toResponse } from "@/server/errors";

/** TOKEN_FILE_EXTENSION is what the Console's file picker filters on. */
export const TOKEN_FILE_EXTENSION = "license";

/** wantsHTML is true for a browser form post, false for an API caller. */
export const wantsHTML = ({ request }: APIContext): boolean =>
  request.headers.get("accept")?.includes("text/html") ?? false;

/** redirect sends a browser back to a portal page with query parameters. */
export const redirect = (
  context: APIContext,
  path: string,
  params: Record<string, string> = {},
): Response => {
  const url = new URL(path, context.url);
  for (const [k, v] of Object.entries(params)) url.searchParams.set(k, v);
  return context.redirect(url.pathname + url.search, 303);
};

/** download answers with the token as a file the Console picker accepts. */
export const download = (token: string, label: string): Response => {
  const stem = label.replace(/[^a-z0-9]+/gi, "-").replace(/^-|-$/g, "") || "synnax";
  return new Response(token, {
    headers: {
      "content-type": "text/plain; charset=utf-8",
      "content-disposition": `attachment; filename="${stem}.${TOKEN_FILE_EXTENSION}"`,
    },
  });
};

/**
 * handle runs a route body and turns a thrown HTTPError into a JSON error for API
 * callers or a redirect carrying the message for a browser form. `back` is the page
 * a browser returns to.
 */
export const handle = async (
  context: APIContext,
  back: string,
  body: () => Promise<Response>,
): Promise<Response> => {
  try {
    return await body();
  } catch (err) {
    if (!wantsHTML(context)) return toResponse(err);
    if (err instanceof HTTPError) {
      if (err.status === 401) return context.redirect("/sign-in", 303);
      return redirect(context, back, { error: err.message });
    }
    console.error(err);
    return redirect(context, back, { error: "Something went wrong. Try again." });
  }
};

const flatten = (value: unknown): string => {
  if (typeof value === "string") return value;
  if (Array.isArray(value)) return value.map(flatten).join(",");
  return JSON.stringify(value) ?? "";
};

/** form reads a posted form body, or JSON for API callers, into one flat record. */
export const form = async ({
  request,
}: APIContext): Promise<Record<string, string>> => {
  const type = request.headers.get("content-type") ?? "";
  if (type.includes("application/json")) {
    const body: unknown = await request.json();
    if (body == null || typeof body !== "object") return {};
    return Object.fromEntries(Object.entries(body).map(([k, v]) => [k, flatten(v)]));
  }
  const data = await request.formData();
  return Object.fromEntries(
    [...data.entries()].map(([k, v]) => [k, typeof v === "string" ? v : v.name]),
  );
};
