// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import React, { cloneElement, HTMLAttributes } from "react";


import IconBlack from "./icon-black.svg";
import IconGradient from "./icon-gradient.svg";
import IconWhite from "./icon-white.svg";
import TitleBlack from "./title-black.svg";
import TitleGradient from "./title-gradient.svg";
import TitleWhite from "./title-white.svg";

export interface LogoProps
  extends Omit<HTMLAttributes<SVGElement>, "width" | "height"> {
  variant?: "icon" | "title";
  color?: "white" | "black" | "gradient" | "auto";
}

const types = {
  "icon-white": <img src={IconWhite} />,
  "icon-black": <img src={IconBlack} />,
  "icon-gradient": <img src={IconGradient} />,
  "title-white": <img src={TitleWhite} />,
  "title-black": <img src={TitleBlack} />,
  "title-gradient": <img src={TitleGradient} />,
};

export const Logo = ({
  variant = "icon",
  color = "auto",
  ...props
}: LogoProps): JSX.Element => {
  let autoColor = color;
  if (color === "auto") {
      autoColor = "gradient";
  }

  const type = `${variant}-${autoColor}`;
  // @ts-expect-error
  const icon = types[type] as React.DetailedReactHTMLElement<any, HTMLElement>;
  return cloneElement(icon, props);
};
