// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, useLayoutEffect } from "react";

import { Client } from "@/client";
import { Aether } from "@/core/aether/main";
import {
  AetherTelemProvider,
  aetherTelemProviderState,
} from "@/telem/TelemProvider/aether";

export interface TelemProviderProps extends PropsWithChildren<any> {}

export const TelemProvider = Aether.wrap<TelemProviderProps>(
  AetherTelemProvider.TYPE,
  ({ children, aetherKey }): ReactElement | null => {
    const [{ path }, , setState] = Aether.use({
      aetherKey,
      type: AetherTelemProvider.TYPE,
      schema: aetherTelemProviderState,
      initialState: {},
    });
    const client = Client.use();

    useLayoutEffect(() => setState({ props: client?.props }), [client, setState]);

    return <Aether.Composite path={path}>{children}</Aether.Composite>;
  }
);
