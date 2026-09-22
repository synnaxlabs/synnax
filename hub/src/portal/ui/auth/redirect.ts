// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const DEFAULT = "/account";

/** target reads where to land after signing in: the page that sent us, or the portal. */
export const target = (): string => {
  const raw = new URLSearchParams(window.location.search).get("redirect_url");
  if (raw == null || !raw.startsWith("/") || raw.startsWith("//")) return DEFAULT;
  return raw;
};

/** withTarget carries the redirect target from this page onto another auth page. */
export const withTarget = (path: string): string => {
  if (typeof window === "undefined") return path;
  const t = target();
  return t === DEFAULT ? path : `${path}?redirect_url=${encodeURIComponent(t)}`;
};
