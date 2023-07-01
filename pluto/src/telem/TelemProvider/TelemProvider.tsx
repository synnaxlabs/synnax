// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, useLayoutEffect } from "react";

import { useClient } from "@/client/ClientContext";
import { Aether } from "@/core/aether/main";
import { Telem, telemState } from "@/telem/TelemProvider/aether";

export interface TelemProviderProps extends PropsWithChildren<any> {}

export const TelemProvider = Aether.wrap<TelemProviderProps>(
  "TelemProvider",
  ({ children, aetherKey }): ReactElement | null => {
    const [{ path }, , setState] = Aether.use({
      aetherKey,
      type: Telem.TYPE,
      schema: telemState,
      initialState: {},
    });
    const client = useClient();

    useLayoutEffect(() => {
      if (client != null) setState({ props: client.props });
    }, [client]);

    return <Aether.Composite path={path}>{children}</Aether.Composite>;
  }
);
