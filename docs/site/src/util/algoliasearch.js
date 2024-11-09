// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import * as dotenv from "dotenv";
import process from "process";
dotenv.config();

import algoliasearch from "algoliasearch";
const client = algoliasearch(
  process.env.DOCS_ALGOLIA_APP_ID,
  process.env.DOCS_ALGOLIA_WRITE_API_KEY,
);

// 1. Build a dataset
import fs from "fs";
import matter from "gray-matter";
import path from "path";
import removeMd from "remove-markdown";

const purgeImports = (content) => {
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

const filenames = fs.readdirSync(path.join("./src/pages"), { recursive: true });
const data = filenames
  .filter((f) => f.endsWith("mdx"))
  .map((filename) => {
    try {
      const markdownWithMeta = fs.readFileSync("./src/pages/" + filename);
      const { data: frontmatter, content } = matter(markdownWithMeta);
      return {
        objectID: filename,
        href: "/" + filename.replace(".mdx", "").replace("index", ""),
        title: frontmatter.heading ?? frontmatter.title,
        description: frontmatter.description,
        content: removeMd(purgeImports(content)).replace(/\n/g, " "),
      };
    } catch (e) {
      console.log(e.message);
    }
  });

const idx = client.initIndex("docs_site");

// delete all objects
await idx.clearObjects();

// 2. Send the dataset in JSON format
const res = await client
  .initIndex("docs_site")
  .saveObjects(JSON.parse(JSON.stringify(data)));

console.log(`Successfully updated ${res.objectIDs.length} pages`);
