// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { MiddlewareHandler } from "astro";

import { Releases } from "@/util/releases";

// The one lookup instance, so one listing serves every page of a build and every
// request of a warm function. `DOCS_GITHUB_TOKEN` raises the GitHub API rate limit.
const releases = new Releases({ token: process.env.DOCS_GITHUB_TOKEN });

export const onRequest: MiddlewareHandler = (context, next) => {
  context.locals.releases = releases;
  return next();
};
