// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, type PropsWithChildren, type ReactElement, use } from "react";

const Context = createContext(false);
Context.displayName = "Context";

/**
 * Removes every copy-link control below it. An app that registers no URL scheme mounts
 * it, because a link it copies can open nothing.
 */
export const Disabled = ({ children }: PropsWithChildren): ReactElement => (
  <Context value>{children}</Context>
);

/** @returns True when links are disabled for this part of the tree. */
export const useDisabled = (): boolean => use(Context);
