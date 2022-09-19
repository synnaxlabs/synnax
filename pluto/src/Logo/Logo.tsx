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
import { ReactComponent as TitleWhite } from "./title-white.svg";
// @ts-ignore
import { ReactComponent as TitleBlack } from "./title-black.svg";
// @ts-ignore
import { ReactComponent as TitleGradient } from "./title-gradient.svg";

import { useThemeContext } from "../Theme/ThemeContext";

export interface LogoProps
  extends Omit<HTMLAttributes<SVGElement>, "width" | "height"> {
  variant?: "icon" | "title";
  color?: "white" | "black" | "gradient" | "auto" | "highContrast";
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
  color = "auto",
  ...props
}: LogoProps) {
  let autoColor: string;
  const { theme } = useThemeContext();
  if (color === "auto")  {
    autoColor = theme.name === "synnax-dark" ? "white" : "gradient";
  } else if (color === "highContrast") {
    autoColor = theme.name === "synnax-dark" ? "white" : "black";
  } else {
    autoColor = color;
  }
  const type = `${variant}-${autoColor}`;
  // @ts-ignore
  const icon = types[type] as React.DetailedReactHTMLElement<any, HTMLElement>;
  return icon ? cloneElement(icon, props) : <h1>Logo Not Found</h1>;
}
