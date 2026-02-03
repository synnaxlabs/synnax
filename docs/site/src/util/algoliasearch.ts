// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import algoliasearch from "algoliasearch";
import * as dotenv from "dotenv";
import fs from "fs";
import matter from "gray-matter";
import path from "path";
import process from "process";
import { remark } from "remark";
import remarkGfm from "remark-gfm";
import stripMarkdown from "strip-markdown";

dotenv.config();

const client = algoliasearch(
  process.env.DOCS_ALGOLIA_APP_ID ?? "",
  process.env.DOCS_ALGOLIA_WRITE_API_KEY ?? "",
);

// 1. Build a dataset

const purgeImports = (content: string): string =>
  content
    // Remove import statements
    .replace(/^import\s+.*?;\s*$/gm, "")
    // Remove export statements
    .replace(/^export\s+.*?;\s*$/gm, "")
    // Remove JSX components (self-closing and with children)
    .replace(/<[A-Z][\w.]*[^>]*\/>/g, "")
    .replace(/<[A-Z][\w.]*[^>]*>[\s\S]*?<\/[A-Z][\w.]*>/g, "")
    .trim();

const toPlainText = async (content: string): Promise<string> => {
  const result = await remark().use(remarkGfm).use(stripMarkdown).process(content);
  return String(result).replace(/\s+/g, " ").trim();
};

const filenames = fs.readdirSync(path.join("./src/pages"), {
  recursive: true,
}) as string[];
const data = await Promise.all(
  filenames
    .filter((f) => f.endsWith("mdx"))
    .map(async (filename) => {
      const markdownWithMeta = fs.readFileSync(`./src/pages/${filename}`);
      const { data: frontmatter, content } = matter(markdownWithMeta);
      let href = `/${filename.replace(".mdx", "").replace("index", "")}`;
      if (filename.includes("releases") && !filename.includes("index"))
        href = `/releases/#${filename
          .replace(".mdx", "")
          .replaceAll("-", "")
          .slice(0, -1)
          .replace("releases/", "")}`;
      return {
        objectID: filename,
        href,
        title: frontmatter.heading ?? frontmatter.title,
        description: frontmatter.description,
        content: await toPlainText(purgeImports(content)),
      };
    }),
);

const idx = client.initIndex("docs_site");

// delete all objects
await idx.clearObjects();

// 2. Send the dataset in JSON format
const res = await client
  .initIndex("docs_site")
  .saveObjects(JSON.parse(JSON.stringify(data)));

console.log(`Successfully updated ${res.objectIDs.length} pages`);
