// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { semver } from "@synnaxlabs/x";
import { glob } from "astro/loaders";
import { defineCollection } from "astro:content";
import { z } from "zod";

// A draft is a .md file, so the collection skips it.
const releases = defineCollection({
  loader: glob({ pattern: "*.mdx", base: "./src/releases" }),
  schema: z.object({
    title: z.string(),
    version: semver.versionZ,
    desktopVersion: semver.versionZ.optional(),
    /** The TypeScript client's version, or null if none shipped. */
    npmVersion: semver.versionZ.nullable().optional(),
    /** Only notes before v0.59 set a date. */
    date: z.string().optional(),
    breakingChanges: z.boolean().default(false),
  }),
});

export const collections = { releases };
