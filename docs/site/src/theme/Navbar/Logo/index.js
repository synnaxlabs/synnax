// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
