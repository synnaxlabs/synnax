// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import favicon from "@synnaxlabs/media/static/logo/icon-white-favicon.ico?inline";
import { type APIRoute } from "astro";

export const prerender = true;

export const GET: APIRoute = async () => {
  const icon = await fetch(favicon);
  return new Response(icon.body, { headers: { "Content-Type": "image/x-icon" } });
};
