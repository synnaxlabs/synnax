// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC, memo as reactMemo } from "react";

/**
 * Memoizes a generic component, keeping its type parameters. React's own `memo`
 * resolves them against their constraints, so its result is callable at that one
 * instantiation only.
 *
 * @example export const Frame = Component.memo(BaseFrame);
 */
export const memo = <T extends FC<any>>(component: T): T =>
  // React types the result as NamedExoticComponent, which shares no overlap with a
  // generic T, so the bridge through unknown is the only way to state the identity.
  reactMemo(component) as unknown as T;
