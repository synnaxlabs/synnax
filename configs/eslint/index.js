module.exports = {
	extends: [
		"eslint:recommended",
		"plugin:@typescript-eslint/recommended",
		"plugin:react/recommended",
		"turbo",
		"prettier",
		"plugin:import/typescript",
		"plugin:react-hooks/recommended",
		"plugin:jsx-a11y/recommended",
		"plugin:testing-library/react",
		"plugin:jest-dom/recommended",
		"plugin:import/errors",
		"plugin:import/warnings",
	],
  "env": {
    "browser": true,
    "amd": true,
    "node": true
  },
	plugins: ["react", "@typescript-eslint"],
	parser: "@typescript-eslint/parser",
	parserOptions: {
		ecmaFeatures: {
			jsx: true,
		},
		ecmaVersion: "latest",
		sourceType: "module",
	},
	settings: {
		react: {
			version: "detect",
		},
		"import/resolver": {
			typescript: {},
		},
	},
	rules: {
		"no-restricted-imports": [
			"error",
			{
				patterns: ["@/features/*/*"],
			},
		],
		"linebreak-style": ["error", "unix"],
		"react/prop-types": "off",

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
					"object",
				],
				"newlines-between": "always",
				alphabetize: { order: "asc", caseInsensitive: true },
			},
		],
		"import/default": "off",
		"import/no-named-as-default-member": "off",
		"import/no-named-as-default": "off",

		"react/react-in-jsx-scope": "off",

		"jsx-a11y/anchor-is-valid": "off",

		"@typescript-eslint/no-unused-vars": ["error"],

		"@typescript-eslint/explicit-function-return-type": ["off"],
		"@typescript-eslint/explicit-module-boundary-types": ["off"],
		"@typescript-eslint/no-empty-function": ["off"],
		"@typescript-eslint/no-explicit-any": ["off"],
	},
};
