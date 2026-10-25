// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { transformSync } from "@babel/core";
import compiler, {
  type CompilerDiagnostic,
  type CompilerErrorDetail,
  LintRules,
  type LoggerEvent,
} from "babel-plugin-react-compiler";
import { type Rule } from "eslint";

import { plugins } from "../compiler";

// Only checks whose errors the configured rules report. An unreported error still stops
// the compiler on its function, which hides every reported error after it.
const LINT = {
  noEmit: true,
  panicThreshold: "none",
  flowSuppressions: false,
  environment: {
    validateRefAccessDuringRender: true,
    validateNoSetStateInRender: true,
    validateNoImpureFunctionsInRender: true,
    validateNoFreezingKnownMutableFunctions: true,
    validateNoVoidUseMemo: true,
    validateHooksUsage: true,
  },
};

const parserPlugins = (filename: string): ("typescript" | "jsx")[] => {
  if (filename.endsWith(".tsx")) return ["typescript", "jsx"];
  if (/\.[mc]?ts$/.test(filename)) return ["typescript"];
  return ["jsx"];
};

interface Options {
  rules: string[];
}

export const reactCompiler: Rule.RuleModule = {
  meta: {
    type: "problem",
    docs: {
      description:
        "Report the errors the React Compiler finds, with the passes that builds run " +
        "before it.",
    },
    schema: [
      {
        type: "object",
        properties: {
          rules: {
            type: "array",
            items: { enum: LintRules.map(({ name }) => name) },
            uniqueItems: true,
          },
        },
        required: ["rules"],
        additionalProperties: false,
      },
    ],
  },
  create: (context) => {
    const [{ rules }] = context.options as [Options];
    const categories = new Set(
      LintRules.filter(({ name }) => rules.includes(name)).map((r) => r.category),
    );
    const { filename, sourceCode } = context;
    const details: (CompilerDiagnostic | CompilerErrorDetail)[] = [];
    const logEvent = (_: string | null, event: LoggerEvent) => {
      if (event.kind === "CompileError") details.push(event.detail);
    };
    transformSync(sourceCode.text, {
      filename,
      babelrc: false,
      configFile: false,
      code: false,
      parserOpts: { plugins: parserPlugins(filename) },
      plugins: plugins([compiler, { ...LINT, logger: { logEvent } }]),
    });
    for (const detail of details) {
      const loc = detail.primaryLocation();
      if (!categories.has(detail.category) || loc == null || typeof loc === "symbol")
        continue;
      context.report({
        loc,
        message: detail.printErrorMessage(sourceCode.text, { eslint: true }),
      });
    }
    return {};
  },
};
