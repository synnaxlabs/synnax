// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import fs from "fs";
import path from "path";

import { render } from "@/theming/static/render";

const headerPath = path.resolve(
  import.meta.dirname,
  "../../../../licenses/headers/template.txt",
);
const headerContent = fs.readFileSync(headerPath, "utf-8").trim();
const currentYear = new Date().getFullYear();
const processedHeader = headerContent.replace(/\{\{YEAR\}\}/g, currentYear.toString());
const COPYRIGHT_HEADER = `/*\n${processedHeader
  .split("\n")
  .map((line) => (line.length > 0 ? ` * ${line}` : " *"))
  .join("\n")}
 */

`;

for (const [file, content] of Object.entries(render()))
  fs.writeFileSync(
    path.join(import.meta.dirname, file),
    COPYRIGHT_HEADER + content,
    "utf-8",
  );
