// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { includeIgnoreFile } from "@eslint/compat";
import pluginJs from "@eslint/js";
import pluginReact2 from "@eslint-react/eslint-plugin";
import type { Linter } from "eslint";
import pluginReact from "eslint-plugin-react";
import simpleImportSort from "eslint-plugin-simple-import-sort";
import globals from "globals";
import path from "path";
import tseslint from "typescript-eslint";
import { fileURLToPath } from "url";

const filename = fileURLToPath(import.meta.url);
const dirname = path.dirname(filename);
const gitignorePath = path.join(dirname, "../../.gitignore");

const config: Linter.Config[] = [
  includeIgnoreFile(gitignorePath),
  { languageOptions: { globals: globals.browser } },
  pluginJs.configs.recommended,
  ...tseslint.configs.recommendedTypeChecked,
  pluginReact.configs.flat.recommended,
  {
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "module",
      globals: { ...globals.browser },
      parserOptions: {
        ecmaFeatures: { jsx: true },
        ecmaVersion: "latest",
        sourceType: "module",
        projectService: true,
        tsconfigRootDir: path.dirname(filename),
      },
    },
    plugins: { "simple-import-sort": simpleImportSort, "@eslint-react": pluginReact2 },
    rules: {
      "no-constant-condition": ["error", { checkLoops: false }],
      "no-duplicate-imports": "error",
      "use-isnan": ["error", { enforceForIndexOf: true }],
      "valid-typeof": ["error", { requireStringLiterals: true }],
      "arrow-body-style": "error",
      curly: ["error", "multi"],
      "dot-notation": "error",
      "logical-assignment-operators": [
        "error",
        "always",
        { enforceForIfStatements: true },
      ],
      "no-array-constructor": "error",
      "no-else-return": ["error", { allowElseIf: false }],
      "no-extra-boolean-cast": ["error", { enforceForInnerExpressions: true }],
      "no-lonely-if": "error",
      "no-new-wrappers": "error",
      "no-object-constructor": "error",
      "no-undef-init": "error",
      "no-unneeded-ternary": ["error", { defaultAssignment: false }],
      "no-useless-computed-key": "error",
      "no-useless-concat": "error",
      "no-useless-constructor": "error",
      "no-useless-rename": "error",
      "no-useless-return": "error",
      "object-shorthand": "error",
      "operator-assignment": "error",
      "prefer-arrow-callback": "error",
      "prefer-const": "error",
      "prefer-exponentiation-operator": "error",
      "prefer-numeric-literals": "error",
      "prefer-object-has-own": "error",
      "prefer-object-spread": "error",
      "prefer-regex-literals": ["error", { disallowRedundantWrapping: true }],
      "prefer-spread": "error",
      "prefer-template": "error",
      radix: ["error", "as-needed"],
      yoda: "error",
      "@typescript-eslint/consistent-type-imports": [
        "error",
        { fixStyle: "inline-type-imports" },
      ],
      "simple-import-sort/imports": "error",
      "simple-import-sort/exports": "error",
      "react/react-in-jsx-scope": "off",
      "react/prop-types": "off",
      "react/jsx-curly-brace-presence": [
        "error",
        { props: "never", children: "never" },
      ],
      "@typescript-eslint/no-empty-object-type": "off",
      "react/no-unescaped-entities": "off",
      "react/jsx-filename-extension": [
        "error",
        { allow: "as-needed", extensions: [".jsx", ".tsx"] },
      ],
      "react/display-name": ["error", { checkContextObjects: true }],
      "react/jsx-boolean-value": "error",
      "react/jsx-no-undef": "error",
      "react/jsx-no-constructed-context-values": "error",
      "react/jsx-no-useless-fragment": "error",
      "@eslint-react/no-context-provider": "error",
      "@eslint-react/no-missing-context-display-name": "error",
      "@typescript-eslint/no-deprecated": "error",
      "@typescript-eslint/no-explicit-any": "off",
      "@typescript-eslint/no-floating-promises": "error",
      "@typescript-eslint/require-await": "off",
      "@typescript-eslint/no-duplicate-type-constituents": "off",
      "@typescript-eslint/no-unsafe-unary-minus": "off",
      "@typescript-eslint/no-unsafe-return": "off",
      "@typescript-eslint/no-unsafe-assignment": "off",
      "@typescript-eslint/no-unsafe-member-access": "off",
      "@typescript-eslint/no-unsafe-call": "off",
      "@typescript-eslint/no-redundant-type-constituents": "off",
      "@typescript-eslint/restrict-plus-operands": "off",
      "@typescript-eslint/no-unsafe-argument": "off",
      "@typescript-eslint/no-unused-vars": [
        "error",
        {
          args: "all",
          argsIgnorePattern: "^_",
          caughtErrors: "all",
          caughtErrorsIgnorePattern: "^_",
          destructuredArrayIgnorePattern: "^_",
          varsIgnorePattern: "^_",
          ignoreRestSiblings: true,
        },
      ],
    },
    settings: {
      "import/resolver": {
        typescript: {}, // this loads <rootdir>/tsconfig.json to eslint
        node: { extensions: [".js", ".jsx", ".ts", ".tsx"] },
      },
      react: { version: "^18.0.0" },
    },
  },
  {
    ignores: [
      "node_modules",
      "build",
      "dist",
      "release",
      "**/*.d.ts",
      "examples",
      "vite.config.ts",
      "stylelint.config.js",
      "bazel-bin",
      "bazel-out",
      "bazel-testlogs",
      "bazel-*",
      "external",
    ],
  },
];

export default config;
