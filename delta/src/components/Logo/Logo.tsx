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

import { Theming } from "@synnaxlabs/pluto";

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
}: LogoProps) => {
  let autoColor = color;
  const { theme } = Theming.useContext();
  if (color == "auto") {
    if (theme.key === "synnax-dark") {
      autoColor = "white";
    } else {
      autoColor = "gradient";
    }
  }

  const type = `${variant}-${autoColor}`;
  // @ts-ignore
  const icon = types[type] as React.DetailedReactHTMLElement<any, HTMLElement>;
  return cloneElement(icon, props);
};
