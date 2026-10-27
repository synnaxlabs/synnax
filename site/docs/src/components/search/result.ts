// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod";

import { Nav } from "@/components/nav";
import { REFERENCE_PAGES } from "@/pages/_nav";

const snippetSchema = z.object({ value: z.string(), matchLevel: z.string() });

/** A page that the docs search index returns for a query. */
export const hitSchema = z.object({
  objectID: z.string(),
  title: z.string(),
  description: z.string().nullish(),
  href: z.string(),
  _snippetResult: z
    .object({ title: snippetSchema.optional(), content: snippetSchema.optional() })
    .optional(),
});

export type Hit = z.infer<typeof hitSchema>;

/** A page that search found, shaped for display. */
export interface Result {
  key: string;
  /** The page's title, with the matched words in `<b>`. */
  title: string;
  /** Text from the page that shows why it matched, with the matched words in `<b>`. */
  excerpt: string;
  href: string;
  /** The name of the section that holds the page. Results show grouped by it. */
  section: string;
  /** The names of the nav groups between the section and the page. */
  path: string[];
  /** The keys of the nav groups that hold the page, outermost first. */
  ancestors: string[];
}

// The index marks each matched word on its own, so a matched phrase joins into one mark.
const joinMarks = (html: string): string => html.replace(/<\/b>(\s+)<b>/g, "$1");

const locate = (href: string): Pick<Result, "section" | "path" | "ancestors"> => {
  if (href.startsWith("/releases"))
    return { section: "Releases", path: [], ancestors: ["releases"] };
  if (href.startsWith("/blog"))
    return { section: "Blog", path: [], ancestors: ["blog"] };
  const chain = Nav.ancestry(REFERENCE_PAGES, href);
  // The index is rebuilt after the site deploys, so it can briefly hold a page that
  // the nav no longer lists.
  if (chain.length === 0) return { section: "Reference", path: [], ancestors: [] };
  return {
    section: chain[0].name,
    path: chain.slice(1, -1).map(({ name }) => name),
    ancestors: chain.flatMap(({ key }) => (key == null ? [] : [key])),
  };
};

/**
 * Shapes a hit for display. A page that matched only by its title shows its
 * description, since the start of its text rarely says what the page is for.
 */
export const fromHit = ({
  objectID,
  title,
  description,
  href,
  _snippetResult,
}: Hit): Result => {
  const content = _snippetResult?.content;
  const excerpt =
    content != null && content.matchLevel !== "none"
      ? content.value
      : (description ?? content?.value ?? "");
  return {
    key: objectID,
    title: joinMarks(_snippetResult?.title?.value ?? title),
    excerpt: joinMarks(excerpt),
    href,
    ...locate(href),
  };
};

/** A result in display order. */
export interface Entry extends Result {
  /** Whether the result is the first of its section, so the section's name shows. */
  first: boolean;
}

/**
 * Orders results by section. A section takes the place of its best result, and its
 * results keep their order.
 */
export const group = (results: Result[]): Entry[] => {
  const sections = new Map<string, Result[]>();
  for (const result of results)
    sections.set(result.section, [...(sections.get(result.section) ?? []), result]);
  return [...sections.values()].flatMap((members) =>
    members.map((result, i) => ({ ...result, first: i === 0 })),
  );
};
