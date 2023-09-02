// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { Aether } from "@/aether";
import { provider } from "@/telem/provider/aether";

export interface TelemProviderProps extends PropsWithChildren<any> {}

export const TelemProvider = Aether.wrap<TelemProviderProps>(
  provider.Provider.TYPE,
  ({ children, aetherKey }): ReactElement | null => {
    const [{ path }] = Aether.use({
      aetherKey,
      type: provider.Provider.TYPE,
      schema: provider.providerStateZ,
      initialState: {},
    });
    return <Aether.Composite path={path}>{children}</Aether.Composite>;
  },
);
