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

import React from "react";
import { Theming, Triggers } from "../src";
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
            <Theming.Provider theme={Theming.themes.synnaxDark}>
                <Triggers.Provider>
                    {Story()}
                </Triggers.Provider>
            </Theming.Provider>
        </React.StrictMode>
    ),
];
