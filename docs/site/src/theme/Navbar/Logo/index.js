import React, { cloneElement, useEffect } from "react";
import Logo from "@theme-original/Navbar/Logo";
import { useColorMode } from "@docusaurus/theme-common";
import {
  applyThemeAsCssVars,
  aryaLight,
  aryaDark,
} from "@arya-analytics/pluto";
import IconGradient from "./icon-gradient.svg";
import IconWhite from "./icon-white.svg";

const types = {
  "icon-white": <IconWhite />,
  "icon-gradient": <IconGradient />,
};

export default function LogoWrapper(props) {
  const { colorMode } = useColorMode();
  useEffect(() => {
    if (colorMode === "dark") {
      applyThemeAsCssVars(document.documentElement, aryaDark);
    } else {
      applyThemeAsCssVars(document.documentElement, aryaLight);
    }
  }, [colorMode]);

  const icon = colorMode === "dark" ? "icon-white" : "icon-gradient";

  return (
    <>
      <a href="/">
        {cloneElement(types[icon], {
          height: 35,
          style: { marginRight: 10, marginTop: 5 },
        })}
      </a>
    </>
  );
}
