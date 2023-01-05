// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FunctionComponent } from "react";

import clsx from "clsx";

import { Button, ButtonProps } from "./Button";
import { ButtonIconOnly } from "./ButtonIconOnly";

import "./ButtonToggle.css";

export interface CoreButtonProps {
  checked: boolean;
}

export const buttonToggleFactory =
  <E extends Pick<ButtonProps, "className" | "variant">>(
    Base: FunctionComponent<E>
  ): FunctionComponent<E & CoreButtonProps> =>
  // eslint-disable-next-line react/display-name
  (props: E & CoreButtonProps): JSX.Element =>
    (
      <Base
        {...props}
        className={clsx(
          "pluto-btn-toggle",
          props.checked && "pluto-btn-toggle--checked",
          props.className
        )}
        variant={props.checked ? props.variant : "outlined"}
      />
    );

export const ButtonToggle = buttonToggleFactory(Button);
ButtonToggle.displayName = "ButtonToggle";

export const ButtonToggleIcon = buttonToggleFactory(ButtonIconOnly);
ButtonToggleIcon.displayName = "ButtonToggleIcon";
