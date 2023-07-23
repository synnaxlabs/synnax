// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FunctionComponent } from "react";

import { CSS } from "@/core/css";
import { Button, ButtonProps } from "@/core/std/Button/Button";
import { ButtonIcon } from "@/core/std/Button/ButtonIcon";
import { InputControl } from "@/core/std/Input";

import "@/core/std/Button/ButtonToggle.css";

const buttonToggleFactory =
  <E extends Pick<ButtonProps, "className" | "variant">>(
    Base: FunctionComponent<E>
  ): FunctionComponent<Omit<E, "value"> & InputControl<boolean>> =>
  // eslint-disable-next-line react/display-name
  ({ value, ...props }) =>
    (
      // @ts-expect-error
      <Base
        {...props}
        checked={value}
        onClick={() => props.onChange(!value)}
        className={CSS(
          CSS.B("btn-toggle"),
          value && CSS.BM("btn-toggle", "checked"),
          props.className
        )}
        variant={value ? props.variant : "outlined"}
      />
    );

export const ButtonToggle = buttonToggleFactory(Button);
ButtonToggle.displayName = "ButtonToggle";

export const ButtonToggleIcon = buttonToggleFactory(ButtonIcon);
ButtonToggleIcon.displayName = "ButtonToggleIcon";
