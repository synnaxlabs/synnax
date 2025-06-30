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
import { type ListenerAdder } from "@/flux/sync/types";

export const AddListenerContext = createContext<ListenerAdder | null>(null);

export const IsStreamerOpenContext = createContext(false);

export const useAddListener = (): ListenerAdder =>
  useRequiredContext(AddListenerContext);

export const useIsStreamerOpen = (): boolean => use(IsStreamerOpenContext);
