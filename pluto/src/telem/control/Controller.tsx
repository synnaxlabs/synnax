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

import { AetherController } from "./aether";

import { Aether } from "@/core";

export interface ControllerProps
  extends z.input<typeof AetherController.stateZ>,
    PropsWithChildren {}

export const Controller = Aether.wrap<ControllerProps>(
  AetherController.TYPE,
  ({ aetherKey, authority, children }) => {
    const [{ path }] = Aether.use({
      aetherKey,
      type: AetherController.TYPE,
      schema: AetherController.stateZ,
      initialState: { authority },
    });
    return <Aether.Composite path={path}>{children}</Aether.Composite>;
  }
);
