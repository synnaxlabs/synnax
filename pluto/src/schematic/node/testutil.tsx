// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { PropsWithChildren, ReactElement } from "react";
import { vi } from "vitest";

import { Enumeration } from "@/enumeration";

/** Gives the forms below it a create prompt that never opens. */
export const CreatePromptStub = ({ children }: PropsWithChildren): ReactElement => (
  <Enumeration.CreatePromptProvider value={vi.fn()}>
    {children}
  </Enumeration.CreatePromptProvider>
);
