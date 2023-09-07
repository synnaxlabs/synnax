// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect } from "react";

import { type z } from "zod";

import { Aether } from "@/aether";
import { Button as Core } from "@/button";
import { useMemoDeepEqualProps } from "@/memo";
import { button } from "@/vis/button/aether";

export interface ButtonProps
  extends Core.ButtonProps,
    Omit<z.input<typeof button.buttonStateZ>, "trigger"> {}

export const Button = Aether.wrap<ButtonProps>(
  button.Button.TYPE,
  ({ aetherKey, sink, ...props }) => {
    const aetherProps = useMemoDeepEqualProps({ sink });

    const [, , setState] = Aether.use({
      aetherKey,
      type: button.Button.TYPE,
      schema: button.buttonStateZ,
      initialState: { trigger: 0, sink },
    });

    useEffect(() => setState((state) => ({ ...state, ...aetherProps })), [aetherProps]);

    const handleClick: Core.ButtonProps["onClick"] = (e) => {
      e.preventDefault();
      setState((state) => ({ ...state, trigger: state.trigger + 1 }));
    };

    return <Core.Button {...props} onClick={handleClick} />;
  },
);
