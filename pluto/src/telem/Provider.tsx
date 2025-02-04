// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { Aether } from "@/aether";
import { telem } from "@/telem/aether";

export interface ProviderProps extends PropsWithChildren<any> {}

export const Provider = ({ children }: ProviderProps): ReactElement => {
  const [{ path }] = Aether.use({
    type: telem.BaseProvider.TYPE,
    schema: telem.providerStateZ,
    initialState: {},
  });
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};
