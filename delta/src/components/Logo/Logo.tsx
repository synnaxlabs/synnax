// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import React, { cloneElement, HTMLAttributes } from "react";

import { Theming } from "@synnaxlabs/pluto";

// @ts-expect-error
import { ReactComponent as IconBlack } from "./icon-black.svg";
// @ts-expect-error
import { ReactComponent as IconGradient } from "./icon-gradient.svg";
// @ts-expect-error
import { ReactComponent as IconWhite } from "./icon-white.svg";
// @ts-expect-error
import { ReactComponent as TitleBlack } from "./title-black.svg";
// @ts-expect-error
import { ReactComponent as TitleGradient } from "./title-gradient.svg";
// @ts-expect-error
import { ReactComponent as TitleWhite } from "./title-white.svg";

export interface LogoProps
  extends Omit<HTMLAttributes<SVGElement>, "width" | "height"> {
  variant?: "icon" | "title";
  color?: "white" | "black" | "gradient" | "auto";
}

const types = {
  "icon-white": <IconWhite />,
  "icon-black": <IconBlack />,
  "icon-gradient": <IconGradient />,
  "title-white": <TitleWhite />,
  "title-black": <TitleBlack />,
  "title-gradient": <TitleGradient />,
};

export const Logo = ({
  variant = "icon",
  color = "auto",
  ...props
}: LogoProps): JSX.Element => {
  let autoColor = color;
  const { theme } = Theming.useContext();
  if (color === "auto") {
    if (theme.key === "synnax-dark") {
      autoColor = "white";
    } else {
      autoColor = "gradient";
    }
  }

  const type = `${variant}-${autoColor}`;
  // @ts-expect-error
  const icon = types[type] as React.DetailedReactHTMLElement<any, HTMLElement>;
  return cloneElement(icon, props);
};
