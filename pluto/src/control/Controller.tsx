// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, useEffect } from "react";

import { type z } from "zod";

import { Aether } from "@/aether";
import { control } from "@/control/aether";
import { useMemoDeepEqualProps } from "@/memo";

export interface ControllerProps
  extends z.input<typeof control.controllerStateZ>,
    PropsWithChildren {
  onStatusChange?: (status: control.Status) => void;
  name: string;
}

export const Controller = Aether.wrap<ControllerProps>(
  control.Controller.TYPE,
  ({ aetherKey, children, onStatusChange, ...props }) => {
    const memoProps = useMemoDeepEqualProps(props);
    const [{ path }, { status }, setState] = Aether.use({
      aetherKey,
      type: control.Controller.TYPE,
      schema: control.controllerStateZ,
      initialState: memoProps,
    });
    useEffect(() => {
      if (status != null) onStatusChange?.(status);
    }, [status, onStatusChange]);
    useEffect(() => setState(memoProps), [memoProps, setState]);

    return <Aether.Composite path={path}>{children}</Aether.Composite>;
  },
);
