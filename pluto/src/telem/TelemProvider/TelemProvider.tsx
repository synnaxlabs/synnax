// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement } from "react";

import { Aether } from "@/aether/main/main";
import { AetherTelemProvider, providerStateZ } from "@/telem/TelemProvider/aether";

export interface TelemProviderProps extends PropsWithChildren<any> {}

export const TelemProvider = Aether.wrap<TelemProviderProps>(
  AetherTelemProvider.TYPE,
  ({ children, aetherKey }): ReactElement | null => {
    const [{ path }] = Aether.use({
      aetherKey,
      type: AetherTelemProvider.TYPE,
      schema: providerStateZ,
      initialState: {},
    });
    return <Aether.Composite path={path}>{children}</Aether.Composite>;
  }
);
