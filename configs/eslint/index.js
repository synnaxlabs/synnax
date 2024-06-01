// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import globals from "globals";
import pluginJs from "@eslint/js";
import tseslint from "typescript-eslint";
import pluginReactConfig from "eslint-plugin-react/configs/recommended.js";
import { fixupConfigRules } from "@eslint/compat";
import simpleImportSort from "eslint-plugin-simple-import-sort";

export default [
    { languageOptions: { globals: globals.browser } },
    pluginJs.configs.recommended,
    ...tseslint.configs.recommended,
    ...fixupConfigRules(pluginReactConfig),
    {
        languageOptions: {
            ecmaVersion: "latest",
            sourceType: "module",
            globals: {
                ...globals.browser,
            },
            parserOptions: {
                ecmaVersion: "latest",
                sourceType: "module",
            },
        },
        ignores: ["node_modules", "node_modules", "build", "dist", "release"],
        plugins: {
            "simple-import-sort": simpleImportSort,
        },
        rules: {
            "simple-import-sort/imports": "error",
            "simple-import-sort/exports": "error",
            // "import/order": [
            //     "error",
            //     {
            //         groups: [
            //             "builtin",
            //             "external",
            //             "internal",
            //             "parent",
            //             "sibling",
            //             "index",
            //             "unknown",
            //         ],
            //         pathGroups: [
            //             {
            //                 pattern: "react",
            //                 group: "external",
            //                 position: "before",
            //             },
            //             {
            //                 pattern: "**/*.css",
            //                 patternOptions: { matchBase: true },
            //                 group: "unknown",
            //                 position: "after",
            //             },
            //         ],
            //         pathGroupsExcludedImportTypes: ["react"],
            //         "newlines-between": "always",
            //         alphabetize: {
            //             order: "asc",
            //             caseInsensitive: true,
            //         },
            //         warnOnUnassignedImports: true,
            //     },
            // ],
            // "no-restricted-imports": [
            //     "error",
            //     {
            //         patterns: [".*"],
            //     },
            // ],
            // "import/no-unresolved": [2],
            // "import/named": "off",
            "react/react-in-jsx-scope": "off",
            // "import/no-default-export": "warn",
            "react/prop-types": "off",
            // "prettier/prettier": ["error", { endOfLine: "auto" }],
            // "@typescript-eslint/no-namespace": "off",
            // "@typescript-eslint/no-confusing-void-expression": "off",
            "@typescript-eslint/ban-types": "off",
            "react/no-unescaped-entities": "off",
            // "@typescript-eslint/non-nullable-type-assertion-style": "off",
            // "@typescript-eslint/no-empty-interface": "off",
            // "@typescript-eslint/ban-ts-comment": "off",
            // "@typescript-eslint/no-dynamic-delete": "off",
            "@typescript-eslint/no-explicit-any": "off",
            "@typescript-eslint/no-unused-vars": [
                "error",
                {
                    "args": "all",
                    "argsIgnorePattern": "^_",
                    "caughtErrors": "all",
                    "caughtErrorsIgnorePattern": "^_",
                    "destructuredArrayIgnorePattern": "^_",
                    "varsIgnorePattern": "^_",
                    "ignoreRestSiblings": true
                  }
            ],
            "no-constant-condition": "off",
        },
        settings: {
            "import/resolver": {
                typescript: {}, // this loads <rootdir>/tsconfig.json to eslint
                node: {
                    extensions: [".js", ".jsx", ".ts", ".tsx"],
                },
            },
            react: {
                version: "^18.0.0",
            },
        },
    },
];
