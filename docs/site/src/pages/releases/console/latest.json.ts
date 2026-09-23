// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type APIRoute } from "astro";

import { manifest } from "@/pages/releases/_manifest";
import { manifestURL } from "@/util/releases";

export const GET: APIRoute = async ({ locals }) =>
  await manifest(locals.releases, "stable", manifestURL);
