// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { MiddlewareHandler } from "astro";
import {
  DOCS_GITHUB_APP_ID,
  DOCS_GITHUB_APP_PRIVATE_KEY,
  GITHUB_TOKEN,
} from "astro:env/server";

import { tokenSource } from "@/util/githubApp";
import { Releases } from "@/util/releases";
import { VERSION } from "@/version";

// The one lookup instance, so one listing serves every page of a build and every
// request of a warm function. The GitHub App raises the GitHub API rate limit. Download
// links keep to the train the docs describe; `next` takes the newest.
const releases = new Releases({
  token: tokenSource({
    appID: DOCS_GITHUB_APP_ID,
    privateKey: DOCS_GITHUB_APP_PRIVATE_KEY,
    token: GITHUB_TOKEN,
  }),
  train: VERSION == null || VERSION.status === "next" ? undefined : VERSION.name,
});

export const onRequest: MiddlewareHandler = (context, next) => {
  context.locals.releases = releases;
  return next();
};
