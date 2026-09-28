// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { navigate } from "astro:transitions/client";

/** RequestError carries the message a portal route answered a request with. */
export class RequestError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "RequestError";
  }
}

const readError = async (res: Response): Promise<RequestError> => {
  let message = res.statusText || "Request failed";
  try {
    const body: unknown = await res.json();
    if (body != null && typeof body === "object" && "error" in body)
      message = String(body.error);
  } catch {
    // A non-JSON body keeps the status text.
  }
  return new RequestError(res.status, message);
};

/** post sends JSON to a portal route and returns its JSON body, or nothing on 204. */
export const post = async <T = void>(path: string, body?: unknown): Promise<T> => {
  const res = await fetch(path, {
    method: "POST",
    headers: {
      accept: "application/json",
      ...(body == null ? {} : { "content-type": "application/json" }),
    },
    body: body == null ? undefined : JSON.stringify(body),
  });
  if (!res.ok) throw await readError(res);
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
};

/** postFile posts to a route that answers with a file and returns it. */
export const postFile = async (path: string): Promise<Blob> => {
  const res = await fetch(path, { method: "POST", headers: { accept: "*/*" } });
  if (!res.ok) throw await readError(res);
  return await res.blob();
};

/** save hands the browser a file to download. */
export const save = (blob: Blob, filename: string): void => {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
};

/** reload re-renders the current page from the server, keeping the URL. */
export const reload = async (): Promise<void> => {
  await navigate(window.location.pathname + window.location.search, {
    history: "replace",
  });
};

/** message reads the text to show for a caught error. */
export const message = (err: unknown): string =>
  err instanceof Error ? err.message : String(err);
