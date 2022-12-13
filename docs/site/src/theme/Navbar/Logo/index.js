import React, { cloneElement, useEffect } from "react";
import { useColorMode } from "@docusaurus/theme-common";
import {
  applyThemeAsCssVars,
  synnaxLight,
  synnaxDark,
} from "@synnaxlabs/pluto";
import "@synnaxlabs/pluto/dist/style.css";
import Logo from "../../../components/Logo/Logo";

export default function LogoWrapper(props) {
  const { colorMode } = useColorMode();
  useEffect(() => {
    applyThemeAsCssVars(
      document.documentElement,
      colorMode === "dark" ? synnaxDark : synnaxLight
    );
  }, [colorMode]);
  return (
    <a href="/" className="navbar__logo">
      <Logo style={{ height: 38 }} {...props} />
    </a>
  );
}
