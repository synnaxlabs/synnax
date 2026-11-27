// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Reads a JSON array of {path, content} from stdin and writes a JSON array of
// {content} or {error} to stdout, in the same order. Each file gets the same result
// as `eslint --fix-dry-run --stdin --stdin-filename <path>` followed by
// `prettier --stdin-filepath <path>`, both run from the current directory.

import { resolve } from "node:path";

import { ESLint } from "eslint";
import * as prettier from "prettier";

const ignorePath = [resolve(".gitignore"), resolve(".prettierignore")];

const lint = async (eslint, path, content) => {
  const [report] = await eslint.lintText(content, {
    filePath: path,
    warnIgnored: true,
  });
  if (report == null) return content;
  const fatal = report.messages.find((m) => m.fatal);
  if (fatal != null) throw new Error(`eslint fatal error: ${fatal.message}`);
  return report.output ?? content;
};

const pretty = async (path, content) => {
  const { ignored } = await prettier.getFileInfo(path, { ignorePath });
  if (ignored) return content;
  const config = await prettier.resolveConfig(path, { editorconfig: true });
  return await prettier.format(content, { ...config, filepath: path });
};

const chunks = [];
for await (const chunk of process.stdin) chunks.push(chunk);
const files = JSON.parse(Buffer.concat(chunks).toString("utf8"));
const eslint = new ESLint({ fix: true });
const results = [];
for (const { path, content } of files)
  try {
    results.push({ content: await pretty(path, await lint(eslint, path, content)) });
  } catch (e) {
    results.push({ error: e instanceof Error ? e.message : String(e) });
  }
process.stdout.write(JSON.stringify(results));
