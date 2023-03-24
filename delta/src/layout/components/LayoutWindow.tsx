// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect } from "react";

import { useSelectWindow, setWindowDecorations } from "@synnaxlabs/drift";
import { Logo } from "@synnaxlabs/media";
import { Nav, Space, useOS, applyCSSVars } from "@synnaxlabs/pluto";
import { appWindow } from "@tauri-apps/api/window";
import { useDispatch } from "react-redux";

import { useSelectLayout } from "../store";

import { LayoutContent } from "./LayoutContent";

import { Controls } from "@/components";
import { CSS } from "@/css";

import "./LayoutWindow.css";

export const NavTop = (): JSX.Element => {
  const os = useOS();
  return (
    <Nav.Bar data-tauri-drag-region location="top" size={"6rem"}>
      <Nav.Bar.Start className="delta-main-nav-top__start">
        <Controls className="delta-controls--macos" visibleIfOS="MacOS" />
        {os === "Windows" && <Logo className="delta-main-nav-top__logo" />}
      </Nav.Bar.Start>
      <Nav.Bar.End>
        <Controls className="delta-controls--windows" visibleIfOS="Windows" />
      </Nav.Bar.End>
    </Nav.Bar>
  );
};

export const LayoutWindow = (): JSX.Element => {
  const { label } = appWindow;
  const win = useSelectWindow(label);
  const layout = useSelectLayout(win?.key ?? "");
  const os = useOS();
  const dispatch = useDispatch();
  useEffect(() => {
    if (os === "Windows") {
      applyWindowsBorders();
      dispatch(setWindowDecorations({ value: false }));
    }
  }, [os]);
  if (layout == null) return <h1>{label}</h1>;
  const content = <LayoutContent layoutKey={layout.key} />;
  return (
    <Space
      empty
      className={CSS(CSS.B("main"), CSS.BM("main", os?.toLowerCase() as string))}
    >
      {layout?.window?.navTop === true && <NavTop />}
      {content}
    </Space>
  );
};

const applyWindowsBorders = (): void => {
  window.document.documentElement.style.boxSizing = "border-box";
  window.document.documentElement.style.border = "var(--pluto-border)";
  window.document.documentElement.style.borderRadius = "var(--pluto-border-radius)";
  applyCSSVars(window.document.documentElement, {
    "--os-border-offset": "2px",
  });
};
