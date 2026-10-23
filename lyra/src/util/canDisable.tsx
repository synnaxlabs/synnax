// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC, type ReactNode } from "react";

export type CanDisabledProps<T extends object & { children?: ReactNode }> = T & {
  disabled?: boolean;
};

export const canDisable = <T extends object & { children?: ReactNode }>(
  C: FC<T>,
): FC<CanDisabledProps<T>> => {
  const O: FC<CanDisabledProps<T>> = ({ disabled = false, ...rest }) =>
    disabled ? rest.children : <C {...(rest as T)} />;
  O.displayName = C.displayName;
  return O;
};
