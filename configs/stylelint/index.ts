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
import postcssHtml from "postcss-html";
import type { Config } from "stylelint";

import { type Baseline, plugins } from "./rules.ts";
import { read } from "./source.ts";

const ROOT = path.resolve(import.meta.dirname, "../..");
const baseline: Baseline = JSON.parse(
  fs.readFileSync(path.join(import.meta.dirname, "baseline.json"), "utf-8"),
);

const config: Config = {
  extends: ["stylelint-config-standard"],
  plugins: plugins({ root: ROOT, source: read(ROOT), baseline }),
  reportDescriptionlessDisables: true,
  reportNeedlessDisables: true,
  rules: {
    // Compares selectors within one file and cannot tell whether two of them ever
    // match one element, so mutually exclusive modifiers flood it.
    "no-descending-specificity": null,
    "selector-max-id": 0,
    "synnax/no-unknown-class": true,
    "synnax/no-unknown-custom-property": true,
    // baseline.json lists the uses from before these two rules. Entries only leave it.
    "synnax/no-foreign-class": true,
    "synnax/no-global-astro-style": true,
    // Task type keys such as modbus_read become modifiers.
    "selector-class-pattern":
      "^[a-z]([a-z0-9-]+)?(__([a-z0-9]+-?)+)?(--([a-z0-9_]+-?)+){0,2}$",
    // Safari needs these two prefixed, and the dev server does not add prefixes.
    "property-no-vendor-prefix": [
      true,
      { ignoreProperties: ["-webkit-user-select", "-webkit-backdrop-filter"] },
    ],
    "declaration-block-no-redundant-longhand-properties": [
      true,
      { ignoreShorthands: ["grid-template"] },
    ],
  },
  overrides: [
    {
      files: ["**/*.astro"],
      customSyntax: postcssHtml,
      rules: {
        "selector-pseudo-class-no-unknown": [true, { ignorePseudoClasses: ["global"] }],
        // postcss-html parses inline style attributes as bare declarations.
        "no-invalid-position-declaration": null,
        "custom-property-empty-line-before": null,
        "declaration-empty-line-before": null,
      },
    },
  ],
};

export default config;
