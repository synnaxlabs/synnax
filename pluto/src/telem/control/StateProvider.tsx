// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { control } from "@/telem/control/aether";

export interface StateProviderProps
  extends z.input<typeof control.stateProviderStateZ>,
    PropsWithChildren {}

export const StateProvider = ({ children }: StateProviderProps): ReactElement => {
  const [{ path }] = Aether.use({
    type: control.StateProvider.TYPE,
    schema: control.stateProviderStateZ,
    initialState: {},
  });
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};
