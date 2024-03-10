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
import { Pluto, Theming } from "../src";
import "./index.css";
import 'reactflow/dist/style.css';

export const parameters = {
    actions: { argTypesRegex: "^on[A-Z].*" },
    controls: {
        matchers: {
            color: /(background|color)$/i,
            date: /Date$/,
        },
    },
};

const CONN_PARAMS = {
    host: "localhost",
    port: 9090,
    username: "synnax",
    password: "seldon"
}



export const decorators = [
    (StoryFn) => (
        <Pluto.Provider
            connParams={CONN_PARAMS} 
        >
            {StoryFn()}
        </Pluto.Provider>
    )
    
];
