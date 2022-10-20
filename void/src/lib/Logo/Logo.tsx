import React, { cloneElement, HTMLAttributes } from "react";
// @ts-ignore
import { ReactComponent as IconGradient } from "./icon-gradient.svg";
// @ts-ignore
import { ReactComponent as IconWhite } from "./icon-white.svg";
// @ts-ignore
import { ReactComponent as IconBlack } from "./icon-black.svg";
// @ts-ignore
import { ReactComponent as TitleWhite } from "./title-white.svg";
// @ts-ignore
import { ReactComponent as TitleBlack } from "./title-black.svg";
// @ts-ignore
import { ReactComponent as TitleGradient } from "./title-gradient.svg";

import { Theme } from "@synnaxlabs/pluto";

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
  const { theme } = Theme.useContext();
  if (mode == "auto") {
    if (theme.name === "synnax-dark") {
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
