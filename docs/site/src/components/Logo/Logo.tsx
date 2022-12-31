// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import React, { cloneElement, HTMLAttributes } from "react";
import IconGradient from "../../../static/img/icon/icon-gradient.svg";
import IconWhite from "../../../static/img/icon/icon-white.svg";
import IconBlack from "../../../static/img/icon/icon-black.svg";
import TitleWhite from "../../../static/img/icon/title-white.svg";
import TitleBlack from "../../../static/img/icon/title-black.svg";
import TitleGradient from "../../../static/img/icon/title-gradient.svg";

import { useColorMode } from "@docusaurus/theme-common";

export interface LogoProps
  extends Omit<HTMLAttributes<SVGElement>, "width" | "height"> {
  variant?: "icon" | "title";
  mode?: "light" | "dark" | "gradient" | "auto";
}

const types = {
  "icon-white": <IconWhite />,
  "icon-black": <IconBlack />,
  "icon-gradient": <IconGradient />,
  "title-white": <TitleWhite />,
  "title-black": <TitleBlack />,
  "title-gradient": <TitleGradient />,
};

export default function Logo({
  variant = "icon",
  mode = "auto",
  ...props
}: LogoProps) {
  let autoVariant: string;
  const { colorMode } = useColorMode();
  if (mode == "auto") {
    if (colorMode === "dark") {
      autoVariant = "white";
    } else {
      autoVariant = "gradient";
    }
  } else {
    autoVariant = variant;
  }
  const type = `${variant}-${autoVariant}`;
  // @ts-ignore
  const icon = types[type] as React.DetailedReactHTMLElement<any, HTMLElement>;
  return cloneElement(icon, props);
}
