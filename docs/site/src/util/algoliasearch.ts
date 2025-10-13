// Copyright 2025 Synnax Labs, Inc.
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
import removeMd from "remove-markdown";

dotenv.config();

const client = algoliasearch(
  process.env.DOCS_ALGOLIA_APP_ID ?? "",
  process.env.DOCS_ALGOLIA_WRITE_API_KEY ?? "",
);

// 1. Build a dataset

const purgeImports = (content: string) => {
  // find the second --- in the file
  const secondDash = content.indexOf("---", 3);
  // get the content after the second ---
  const nc = content.slice(secondDash + 2);
  // find the first markdown header in the file
  const firstHeader = nc.indexOf("#");
  // find the first 'import' statement in the file
  const firstImport = nc.indexOf("import");
  // find the index of the first newline after the first import statement
  if (firstImport > firstHeader || firstImport === -1) return nc;
  // find the index of the last import statement before the first markdown header
  const lastImport = nc.slice(0, firstHeader).lastIndexOf("import");
  const lastNewline = nc.slice(lastImport + 1).indexOf("\n");
  // return the content with the imports removed
  return nc.slice(lastImport + lastNewline + 2);
};

const filenames = fs.readdirSync(path.join("./src/pages"), {
  recursive: true,
}) as string[];
const data = filenames
  .filter((f) => f.endsWith("mdx"))
  .map((filename) => {
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
      content: removeMd(purgeImports(content)).replace(/\n/g, " "),
    };
  });

const idx = client.initIndex("docs_site");

// delete all objects
await idx.clearObjects();

// 2. Send the dataset in JSON format
const res = await client
  .initIndex("docs_site")
  .saveObjects(JSON.parse(JSON.stringify(data)));

console.log(`Successfully updated ${res.objectIDs.length} pages`);
