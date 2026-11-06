// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useSyncExternalStore } from "react";

import { type Clock, clock } from "@/ui/format";

const BROWSER = clock();
const SERVER = clock("UTC");

const subscribe = (): (() => void) => () => {};

const browser = (): Clock => BROWSER;

const server = (): Clock => SERVER;

/**
 * useClock returns a clock in the browser's time zone. The server render and hydration
 * read UTC, then the component renders again in the browser's zone.
 */
export const useClock = (): Clock => useSyncExternalStore(subscribe, browser, server);
