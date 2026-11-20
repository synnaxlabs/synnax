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

import { CLIENTS } from "@/components/client/choice";
import { PLATFORMS } from "@/components/platform/choice";

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

/**
 * Loads the pages under a directory of src/docs. Each id is the page's route. A file
 * that starts with an underscore is a partial, not a page.
 */
const pages = (dir: string) =>
  glob({
    pattern: [`${dir}/**/*.mdx`, "!**/_*"],
    base: "./src/docs",
    generateId: ({ entry }) => entry.replace(/\.mdx$/, "").replace(/\/index$/, ""),
  });

const pageZ = z.object({
  title: z.string(),
  /** The page heading, when it differs from the title. */
  heading: z.string().optional(),
  description: z.string(),
});

const reference = defineCollection({
  loader: pages("reference"),
  schema: pageZ.extend({
    clients: z.array(z.enum(CLIENTS)).optional(),
    platforms: z.array(z.enum(PLATFORMS)).optional(),
    breadcrumbHidden: z.boolean().default(false),
    onThisPageHidden: z.boolean().default(false),
  }),
});

const blog = defineCollection({
  loader: pages("blog"),
  schema: pageZ.extend({
    published: z.coerce.date(),
    author: z.string().optional(),
    authorImage: z.url().optional(),
    authorLink: z.url().optional(),
  }),
});

export const collections = { releases, reference, blog };
