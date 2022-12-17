const path = require("path");
const tsconfigPaths = require("vite-tsconfig-paths").default;

console.log(path.resolve(path.dirname(__dirname), "tsconfig.json").toString());

module.exports = {
    stories: ["../src/**/*.stories.mdx", "../src/**/*.stories.@(js|jsx|ts|tsx)"],
    addons: [
        "@storybook/addon-links",
        "@storybook/addon-essentials",
        "@storybook/addon-interactions",
    ],
    framework: "@storybook/react",
    core: {
        builder: "@storybook/builder-vite",
    },
    features: {
        storyStoreV7: true,
    },
    resolve: {},
    async viteFinal(config) {
        config.plugins.push(
            tsconfigPaths({
                projects: [path.resolve(path.dirname(__dirname), "tsconfig.json")],
            })
        );
        return config;
    },
};
