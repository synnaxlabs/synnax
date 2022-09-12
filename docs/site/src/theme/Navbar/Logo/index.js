import React, { cloneElement, useEffect } from "react";
import Logo from "@theme-original/Navbar/Logo";
import { useColorMode } from "@docusaurus/theme-common";
import {
  applyThemeAsCssVars,
  aryaLight,
  aryaDark,
} from "@synnaxlabs/pluto";
import IconGradient from "../../../../static/img/icon-gradient.svg";
import IconWhite from "../../../../static/img/icon-white.svg";
import "@synnaxlabs/pluto/dist/style.css";

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
    <a href="/" {...props} className="navbar__logo">
      {cloneElement(types[icon], {
        height: 36,
      })}
    </a>
  );
}
