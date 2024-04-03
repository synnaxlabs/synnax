// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PageNavNode } from "@/components/PageNav";

export const typescriptClientNav: PageNavNode = {
    key: "typescript-client",
    name: "Typescript Client",
    children: [
        {
            key: "/typescript-client/get-started",
            href: "/typescript-client/get-started",
            name: "Get Started",
        },
        {
            key: "/typescript-client/channels",
            href: "/typescript-client/channels",
            name: "Channels",
        },
        {
            key: "/typescript-client/read-telemetry",
            href: "/typescript-client/read-telemetry",
            name: "Read Telemetry",
        },
        {
            key: "/typescript-client/write-telemetry",
            href: "/typescript-client/write-telemetry",
            name: "Write Telemetry",
        },
    ]
}