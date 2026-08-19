// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { Button, Icon, Theming } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { Theme } from "@/session/theme";

export interface ToggleProps extends Omit<Button.ButtonProps, "onClick" | "children"> {}

/** Icon-only button that flips the color theme opposite the effective one. */
export const Toggle = (props: ToggleProps): ReactElement => {
  const dark = Theming.use().key === Theming.SYNNAX_DARK.key;
  const dispatch = useDispatch<Dispatch<Theme.Action>>();
  const next = dark ? "light" : "dark";
  return (
    <Button.Button
      variant="text"
      tooltip={`Switch to ${next} theme`}
      onClick={() => dispatch(Theme.set(next))}
      {...props}
    >
      {dark ? <Icon.LightMode /> : <Icon.DarkMode />}
    </Button.Button>
  );
};
