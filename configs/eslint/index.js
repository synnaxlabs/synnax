/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

module.exports = {
    env: {
        browser: true,
        es2021: true,
    },
    extends: [
        "plugin:react/recommended",
        "standard-with-typescript",
        "plugin:import/recommended",
        "prettier",
    ],
    overrides: [],
    parserOptions: {
        ecmaVersion: "latest",
        sourceType: "module",
    },
    plugins: ["import", "react", "prettier"],
    rules: {
        "import/order": [
            "error",
            {
                groups: [
                    "builtin",
                    "external",
                    "internal",
                    "parent",
                    "sibling",
                    "index",
                    "unknown"
                ],
                pathGroups: [
                    {
                        pattern: "react",
                        group: "external",
                        position: "before",
                    },
                    {
                        pattern: "**/*.css",
                        patternOptions: { matchBase: true },
                        group: "unknown",
                        position: "after",
                    },
                ],
                pathGroupsExcludedImportTypes: ["react"],
                "newlines-between": "always",
                alphabetize: {
                    order: "asc",
                    caseInsensitive: true,
                },
                warnOnUnassignedImports: true,
            },
        ],
        "import/no-unresolved": [
            2,
            // {
            //     ignore: ["^@/"],
            //     commonjs: true,
            //     amd: true,
            // },
        ],
        "import/named": "off",
        "react/react-in-jsx-scope": "off",
        "import/no-default-export": "warn",
        "react/prop-types": "off",
        "prettier/prettier": "error",
        "@typescript-eslint/no-namespace": "off",
        "@typescript-eslint/no-confusing-void-expression": "off",
        "@typescript-eslint/ban-types": "off",
    },
    settings: {
        "import/resolver": {
            typescript: {}, // this loads <rootdir>/tsconfig.json to eslint
            node: {
                extensions: [".js", ".jsx", ".ts", ".tsx"],
            },
        },
        react: {
            version: "detect",
        },
    },
};
