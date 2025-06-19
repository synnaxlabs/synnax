// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, use } from "react";

import { useRequiredContext } from "@/hooks";
import { type ListenerAdder } from "@/sync/types";

export const Context = createContext<ListenerAdder | null>(null);

export const TestContext = createContext(false);

export const useAddListener = (): ListenerAdder => useRequiredContext(Context);

export const useStreamerIsOpen = (): boolean => use(TestContext);
