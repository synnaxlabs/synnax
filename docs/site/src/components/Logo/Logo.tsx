import React, {
  cloneElement,
  HTMLAttributes,
} from "react";
import IconGradient from "./icon-gradient.svg";
import IconWhite from "./icon-white.svg";
// @ts-ignore
import IconBlack from "./icon-black.svg";
// @ts-ignore
import TitleWhite from "./title-white.svg";
// @ts-ignore
import TitleBlack from "./title-black.svg";
// @ts-ignore
import TitleGradient from "./title-gradient.svg";

import { useThemeContext } from "@synnaxlabs/pluto";
import {useColorMode} from "@docusaurus/theme-common";

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
