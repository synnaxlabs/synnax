// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIContext } from "astro";

import { toResponse } from "@/server/errors";

/** TOKEN_FILE_EXTENSION is what the Console's file picker filters on. */
export const TOKEN_FILE_EXTENSION = "license";

/** filename is the name a token downloads as, derived from its license label. */
export const filename = (label: string): string => {
  const stem = label.replace(/[^a-z0-9]+/gi, "-").replace(/^-|-$/g, "") || "synnax";
  return `${stem}.${TOKEN_FILE_EXTENSION}`;
};

/** download answers with the token as a file the Console picker accepts. */
export const download = (token: string, label: string): Response =>
  new Response(token, {
    headers: {
      "content-type": "text/plain; charset=utf-8",
      "content-disposition": `attachment; filename="${filename(label)}"`,
    },
  });

/** handle runs a route body and turns a thrown error into a JSON error response. */
export const handle = async (body: () => Promise<Response>): Promise<Response> => {
  try {
    return await body();
  } catch (err) {
    return toResponse(err);
  }
};

const flatten = (value: unknown): string => {
  if (typeof value === "string") return value;
  if (Array.isArray(value)) return value.map(flatten).join(",");
  return JSON.stringify(value) ?? "";
};

/** form reads a JSON body, or a posted form, into one flat record. */
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
