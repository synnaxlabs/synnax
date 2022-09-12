import React, {
  cloneElement,
  HTMLAttributes,
} from "react";
// @ts-ignore
import { ReactComponent as IconGradient } from "./icon-gradient.svg";
// @ts-ignore
import { ReactComponent as IconWhite } from "./icon-white.svg";
// @ts-ignore
import { ReactComponent as IconBlack } from "./icon-black.svg";
// @ts-ignore
import { ReactComponent as TitleWhite } from "./full-white.svg";
// @ts-ignore
import { ReactComponent as TitleBlack } from "./full-black.svg";
// @ts-ignore
import { ReactComponent as TitleGradient } from "./full-gradient.svg";

import { useThemeContext } from "@arya-analytics/pluto";

export interface LogoProps
  extends Omit<HTMLAttributes<SVGElement>, "width" | "height"> {
  variant?: "icon" | "title";
  mode?: "light" | "dark" | "gradient" | "auto";
}

const types = {
  "icon-white": <IconWhite />,
  "icon-black": <IconBlack />,
  "icon-gradient": <IconGradient />,
  "title-white": <FullWhite />,
  "title-black": <FullBlack />,
  "title-gradient": <FullGradient />,
};

export default function Logo({
  variant = "icon",
  mode = "auto",
  ...props
}: LogoProps) {
  let autoVariant: string;
  const { theme } = useThemeContext();
  if (mode == "auto") {
    if (theme.name === "arya-dark") {
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
