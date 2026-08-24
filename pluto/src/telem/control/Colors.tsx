// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { Aether } from "@/aether";
import { control } from "@/telem/control/aether";

export interface ColorsProps extends PropsWithChildren {}

export const Colors = ({ children }: ColorsProps): ReactElement => {
  const [{ path }] = Aether.use({
    type: control.Colors.TYPE,
    schema: control.colorsStateZ,
    initialState: {},
  });
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};
