// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import pluginJs from "@eslint/js";
import pluginReact from "@eslint-react/eslint-plugin";
import stylistic from "@stylistic/eslint-plugin";
import type { ESLint, Linter, Rule } from "eslint";
import { includeIgnoreFile } from "eslint/config";
import reactHooks from "eslint-plugin-react-hooks";
import simpleImportSort from "eslint-plugin-simple-import-sort";
import globals from "globals";
import path from "path";
import tseslint from "typescript-eslint";

import { isWorker } from "./compiler";
import { barrelImports } from "./rules/barrelImports";
import { jsxFilenameExtension } from "./rules/jsxFilenameExtension";
import { namedHookCallers } from "./rules/namedHookCallers";
import { namedHookValues } from "./rules/namedHookValues";
import { reactCompiler } from "./rules/reactCompiler";

// ESLint matches files only by glob, so a rule skips Aether worker code itself.
const skipWorkers = (rule: Rule.RuleModule): Rule.RuleModule => ({
  ...rule,
  create: (context) => (isWorker(context.filename) ? {} : rule.create(context)),
});

const synnax: ESLint.Plugin = {
  rules: {
    "barrel-imports": barrelImports,
    "jsx-filename-extension": jsxFilenameExtension,
    "named-hook-callers": skipWorkers(namedHookCallers),
    "named-hook-values": skipWorkers(namedHookValues),
    "react-compiler": skipWorkers(reactCompiler),
  },
};

const COMPILER_RULES: Linter.RulesRecord = {
  "react-hooks/rules-of-hooks": "error",
  "synnax/react-compiler": [
    "error",
    {
      rules: [
        "hooks",
        "use-memo",
        "preserve-manual-memoization",
        "incompatible-library",
        "immutability",
        "globals",
        "refs",
        "purity",
        "set-state-in-render",
        "component-hook-factories",
        "unsupported-syntax",
        "todo",
        "syntax",
        "invariant",
        "config",
        "gating",
      ],
    },
  ],
  "synnax/named-hook-callers": "error",
  "synnax/named-hook-values": "error",
};

const NAMESPACE_IMPORT = {
  selector: "ImportDeclaration[source.value=/^(@\\/|\\.)/] > ImportNamespaceSpecifier",
  message:
    "Import a local module's namespace from its barrel, which forms it with `export * as`.",
};

const RESTRICTED_SYNTAX = [
  {
    selector: "JSXAttribute > JSXExpressionContainer > Literal.expression[value=true]",
    message: "Write a true boolean attribute without a value.",
  },
  NAMESPACE_IMPORT,
];

const MEMO_MESSAGE =
  "The React Compiler caches values and functions. Write a plain value or function.";

/** Bans manual useMemo and useCallback calls in a package built with the compiler. */
export const reactCompiled: Linter.Config = {
  rules: {
    // A later config replaces the rule's whole selector list, so this one repeats it.
    "no-restricted-syntax": [
      "error",
      ...RESTRICTED_SYNTAX,
      {
        selector: "CallExpression[callee.name=/^use(Memo|Callback)$/]",
        message: MEMO_MESSAGE,
      },
      {
        selector:
          "CallExpression[callee.object.name='React'][callee.property.name=/^use(Memo|Callback)$/]",
        message: MEMO_MESSAGE,
      },
    ],
  },
};

const gitignorePath = path.join(import.meta.dirname, "../../.gitignore");

const reactPreset = pluginReact.configs["recommended-type-checked"];

const config: Linter.Config[] = [
  includeIgnoreFile(gitignorePath),
  { languageOptions: { globals: globals.browser } },
  pluginJs.configs.recommended,
  ...tseslint.configs.recommendedTypeChecked,
  {
    ...reactPreset,
    // The preset warns on most rules, and a warning does not fail lint.
    rules: Object.fromEntries(
      Object.keys(reactPreset.rules ?? {}).map((rule) => [rule, "error" as const]),
    ),
  },
  {
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "module",
      globals: { ...globals.browser },
      parserOptions: {
        ecmaVersion: "latest",
        sourceType: "module",
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      "simple-import-sort": simpleImportSort,
      "@stylistic": stylistic,
      "react-hooks": {
        rules: {
          ...reactHooks.rules,
          "rules-of-hooks": skipWorkers(reactHooks.rules["rules-of-hooks"]),
        },
      },
      synnax,
    },
    rules: {
      "synnax/barrel-imports": "error",
      "synnax/jsx-filename-extension": "error",
      "new-cap": ["error", { capIsNew: false }],
      "no-constant-condition": ["error", { checkLoops: false }],
      "no-duplicate-imports": "error",
      "no-loss-of-precision": "error",
      "use-isnan": ["error", { enforceForIndexOf: true }],
      "preserve-caught-error": ["error", { requireCatchParameter: true }],
      "valid-typeof": ["error", { requireStringLiterals: true }],
      "arrow-body-style": "error",
      curly: ["error", "multi"],
      "dot-notation": "error",
      // Allows the if-statement form: the React Compiler cannot compile ??= or ||=.
      "logical-assignment-operators": ["error", "always"],
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
      radix: "error",
      yoda: "error",
      "@typescript-eslint/consistent-type-imports": [
        "error",
        { fixStyle: "inline-type-imports" },
      ],
      "simple-import-sort/imports": "error",
      "simple-import-sort/exports": "error",
      "@stylistic/jsx-curly-brace-presence": [
        "error",
        { props: "never", children: "never" },
      ],
      "no-restricted-syntax": ["error", ...RESTRICTED_SYNTAX],
      "@eslint-react/no-missing-component-display-name": "error",
      "@eslint-react/no-missing-context-display-name": "error",
      "@eslint-react/jsx-no-useless-fragment": "error",
      "@eslint-react/dom-no-unsafe-target-blank": "error",
      // Allows force-update hooks and acronym setters such as setOAuthError.
      "@eslint-react/use-state": [
        "error",
        { enforceAssignment: false, enforceSetterName: false },
      ],
      // Off until a SY-5149 hook audit: the fixes change render behavior.
      "@eslint-react/exhaustive-deps": "off",
      "@eslint-react/set-state-in-effect": "off",
      // Off: the rules below check these with the React Compiler.
      "@eslint-react/rules-of-hooks": "off",
      "@eslint-react/purity": "off",
      // Off: matches our own useContext hooks from context.create by name.
      "@eslint-react/no-use-context": "off",
      // Off: flags a call that returns a primitive key.
      "@eslint-react/no-unstable-context-value": "off",
      // Off: flags stable components read from a registry.
      "@eslint-react/static-components": "off",
      // Off: our nested components are memoized on purpose.
      "@eslint-react/no-nested-component-definitions": "off",
      // Off: false hits on Select.useMemo, which takes a selector.
      "@eslint-react/use-memo": "off",
      // Off: false hits on hook slots (useVisible) and Aether use functions.
      "@eslint-react/no-unnecessary-use-prefix": "off",
      // Off: our index keys are on static lists with no stable ID.
      "@eslint-react/no-array-index-key": "off",
      // Off: Lyra primitives use these APIs on purpose.
      "@eslint-react/no-children-count": "off",
      "@eslint-react/no-children-for-each": "off",
      "@eslint-react/no-children-map": "off",
      "@eslint-react/no-children-to-array": "off",
      "@eslint-react/no-clone-element": "off",
      "@eslint-react/dom-no-flush-sync": "off",
      // Off: the docs and site render trusted, pre-built HTML.
      "@eslint-react/dom-no-dangerously-set-innerhtml": "off",
      // Off: flags try blocks in specs that render nothing inside them.
      "@eslint-react/error-boundaries": "off",
      // Off: names follow our namespace rules, not fixed suffixes.
      "@eslint-react/naming-convention-context-name": "off",
      "@eslint-react/naming-convention-id-name": "off",
      "@eslint-react/naming-convention-ref-name": "off",
      "@typescript-eslint/no-deprecated": "error",
      "@typescript-eslint/use-unknown-in-catch-callback-variable": "error",
      "@typescript-eslint/only-throw-error": [
        "error",
        {
          allowThrowingAny: false,
          allowThrowingUnknown: false,
          allowRethrowing: false,
        },
      ],
      "@typescript-eslint/no-explicit-any": "off",
      "@typescript-eslint/no-floating-promises": ["error", { checkThenables: true }],
      "@typescript-eslint/no-meaningless-void-operator": "error",
      "@typescript-eslint/no-misused-promises": [
        "error",
        { checksConditionals: { flagUnions: "strict" } },
      ],
      "@typescript-eslint/no-empty-object-type": [
        "error",
        { allowInterfaces: "with-single-extends", allowObjectTypes: "always" },
      ],
      "@typescript-eslint/no-mixed-enums": "error",
      "@typescript-eslint/no-unsafe-enum-assignment": "error",
      "@typescript-eslint/unified-signatures": "error",
      "@typescript-eslint/require-await": "off",
      "@typescript-eslint/no-duplicate-type-constituents": "off",
      "@typescript-eslint/return-await": ["error", "always"],
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
    },
  },
  {
    // Oracle imports one generated module into another as a namespace.
    files: ["**/*.gen.ts"],
    rules: {
      "no-restricted-syntax": [
        "error",
        ...RESTRICTED_SYNTAX.filter((r) => r !== NAMESPACE_IMPORT),
      ],
    },
  },
  {
    // The React Compiler skips a component or hook that breaks one of these rules.
    // Specs do not ship, so their harnesses may break them.
    ignores: ["**/*.spec.ts", "**/*.spec.tsx"],
    rules: COMPILER_RULES,
  },
  {
    // Packages declare their JavaScript free of side effects, so a bundler skips a
    // barrel's own body and drops any stylesheet it imports.
    files: ["**/index.ts", "**/index.tsx", "**/external.ts", "**/external.tsx"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["*.css"],
              message: "Import a stylesheet from the component that uses it.",
            },
          ],
        },
      ],
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
      "vitest.bench.config.ts",
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
