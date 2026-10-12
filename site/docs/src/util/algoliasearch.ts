// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { algoliasearch } from "algoliasearch";
import fs from "fs";
import matter from "gray-matter";
import path from "path";
import process from "process";
import { remark } from "remark";
import remarkGfm from "remark-gfm";
import stripMarkdown from "strip-markdown";

import * as releaseNotes from "./releaseNotes.ts";

const client = algoliasearch(
  process.env.DOCS_ALGOLIA_APP_ID ?? "",
  process.env.DOCS_ALGOLIA_WRITE_API_KEY ?? "",
);

const purgeImports = (content: string): string =>
  content
    .replace(/^import\s+.*?;\s*$/gm, "")
    .replace(/^export\s+.*?;\s*$/gm, "")
    // Remove JSX components (self-closing and with children)
    .replace(/<[A-Z][\w.]*[^>]*\/>/g, "")
    .replace(/<[A-Z][\w.]*[^>]*>[\s\S]*?<\/[A-Z][\w.]*>/g, "")
    .trim();

const toPlainText = async (content: string): Promise<string> => {
  const result = await remark().use(remarkGfm).use(stripMarkdown).process(content);
  return String(result).replace(/\s+/g, " ").trim();
};

// A page behind a flag is indexed only when the flag is on, read from the same
// FLAG_<NAME> variables the site build uses.
const hidden = (flag: unknown): boolean =>
  typeof flag === "string" && process.env[`FLAG_${flag.toUpperCase()}`] !== "true";

const read = (dir: string, filename: string, href: string) => ({
  filename,
  href,
  ...matter(fs.readFileSync(path.join(dir, filename))),
});

const PAGES_DIR = "./src/pages";
const pages = [
  ...fs
    .readdirSync(PAGES_DIR, { recursive: true, encoding: "utf8" })
    .filter((f) => f.endsWith("mdx"))
    .map((f) =>
      read(PAGES_DIR, f, `/${f.replace(".mdx", "").replace(/(^|\/)index$/, "$1")}`),
    ),
  // Each version's notes render only inside /releases, under the version's anchor.
  ...releaseNotes
    .list()
    .map(({ name, anchor }) =>
      read(releaseNotes.DIR, `${name}.mdx`, `/releases/#${anchor}`),
    ),
];
const skipped = pages.filter(({ data }) => hidden(data.flag));
const data = await Promise.all(
  pages
    .filter(({ data }) => !hidden(data.flag))
    .map(async ({ filename, href, data: frontmatter, content }) => ({
      objectID: filename,
      href,
      title: frontmatter.heading ?? frontmatter.title,
      description: frontmatter.description,
      content: await toPlainText(purgeImports(content)),
    })),
);

console.log(`Indexing ${data.length} pages, skipping ${skipped.length} behind flags`);

await client.clearObjects({ indexName: "docs_site" });

const res = await client.saveObjects({ indexName: "docs_site", objects: data });

console.log(`Successfully updated ${res.length} pages`);
