// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC, type PropsWithoutRef } from "react";

export const createComponent = <P extends PropsWithoutRef<any>>(
  Base: FC<P>,
  props: Partial<P>,
): FC<P> => {
  const OC: FC<P> = (p) => <Base {...props} {...p} />;
  OC.displayName = Base.displayName;
  return OC;
};
