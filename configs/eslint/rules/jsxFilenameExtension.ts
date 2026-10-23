// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Rule } from "eslint";

export const jsxFilenameExtension: Rule.RuleModule = {
  meta: {
    type: "suggestion",
    docs: { description: "Require JSX in a .tsx file." },
    messages: { noJSX: "Rename this file to .ts, because it has no JSX." },
    schema: [],
  },
  create: (context) => {
    if (!context.filename.endsWith(".tsx")) return {};
    let found = false;
    const mark = (): void => {
      found = true;
    };
    return {
      JSXElement: mark,
      JSXFragment: mark,
      "Program:exit": (node) => {
        if (found || node.loc == null) return;
        context.report({ node, loc: node.loc.end, messageId: "noJSX" });
      },
    };
  },
};
