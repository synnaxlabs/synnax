// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren } from "react";

import { z } from "zod";

import { Aether } from "@/aether";
import { control } from "@/telem/control/aether";

export interface ControllerProps
  extends z.input<typeof control.controllerStateZ>,
    PropsWithChildren {}

export const Controller = Aether.wrap<ControllerProps>(
  control.Controller.TYPE,
  ({ aetherKey, authority, children }) => {
    const [{ path }] = Aether.use({
      aetherKey,
      type: control.Controller.TYPE,
      schema: control.controllerStateZ,
      initialState: { authority },
    });
    return <Aether.Composite path={path}>{children}</Aether.Composite>;
  }
);
