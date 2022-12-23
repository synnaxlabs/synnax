module.exports = {
    root: true,
    extends: ["synnaxlabs"],
    parserOptions: {
        project: "./tsconfig.json",
    },
    rules: {
        "no-restricted-imports": [
            "error",
            {
                patterns: ["@/features/*/*", "@/hooks/*/*", "@/components/*/*"],
            },
        ],
    },
};
