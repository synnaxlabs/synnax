import React from "react";
import { Theming } from "../src";
import "./index.css";

export const parameters = {
    actions: { argTypesRegex: "^on[A-Z].*" },
    controls: {
        matchers: {
            color: /(background|color)$/i,
            date: /Date$/,
        },
    },
};

export const decorators = [
    (Story) => (
        <React.StrictMode>
            <Theming.Provider theme={Theming.themes.synnaxDark}>{Story()}</Theming.Provider>
        </React.StrictMode>
    ),
];
