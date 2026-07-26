// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Icon, Nav, OS } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { Toolbars } from "@/app/toolbars";
import { Cluster } from "@/feature/cluster";
import { Docs } from "@/feature/docs";
import { Panel } from "@/feature/panel";
import { Project } from "@/feature/project";
import { Version } from "@/feature/version";
import { CSS } from "@/platform/css";
import { Nav as PlatformNav } from "@/platform/nav";
import { User } from "@/platform/user";
import { Window } from "@/platform/window";
import { Session } from "@/session";

const BottomToggleButton = (): ReactElement => {
  const dispatch = Session.useDispatch();
  const toggle = useCallback(() => dispatch(Session.Nav.toggleBottom({})), []);
  return (
    <Button.Button
      variant="outlined"
      className={CSS.BE("mosaic", "controls-button")}
      onClick={toggle}
      justify="center"
      size="small"
      contrast={2}
      color={9}
      weight={450}
      triggerIndicator={Toolbars.BOTTOM.trigger}
    >
      <Icon.Visualize />
      Controls
    </Button.Button>
  );
};

export interface TopProps {
  /** Renders the trimmed secondary window variant: the controls toggle in place of
   * the project selector and badge cluster. */
  secondary?: boolean;
}

export const Top = ({ secondary = false }: TopProps): ReactElement => {
  const os = OS.use();
  return (
    <PlatformNav.Bar location="top" size="6.5rem" data-tauri-drag-region>
      <Nav.Bar.Start data-tauri-drag-region gap="large">
        <Window.Controls visibleIfOS="macOS" forceOS={os} />
        {secondary ? os === "Windows" && <BottomToggleButton /> : <Project.Selector />}
        <Panel.Selector />
      </Nav.Bar.Start>
      <Nav.Bar.Content data-tauri-drag-region full="x" />
      <Nav.Bar.End justify="end" align="center" data-tauri-drag-region gap="small">
        {secondary ? (
          os === "macOS" && <BottomToggleButton />
        ) : (
          <>
            <Version.Badge />
            <User.Badge />
            <Cluster.ConnectionBadge />
            <Docs.OpenButton />
          </>
        )}
        <Window.Controls visibleIfOS="Windows" forceOS={os} />
      </Nav.Bar.End>
    </PlatformNav.Bar>
  );
};
