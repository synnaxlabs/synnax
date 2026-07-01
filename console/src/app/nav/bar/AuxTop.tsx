// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logo } from "@synnaxlabs/media";
import { Button, Icon, Nav, OS, Project as PProject, Text } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Items } from "@/app/nav/items";
import { CSS } from "@/primitive/css";
import { Nav as ComponentNav } from "@/primitive/nav";
import { Window } from "@/primitive/window";
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
      triggerIndicator={Items.BOTTOM.trigger}
    >
      <Icon.Visualize />
      Controls
    </Button.Button>
  );
};

export const AuxTop = (): ReactElement => {
  const os = OS.use();
  const activeName = Session.Layout.useSelectActiveMosaicTabName();
  const activeProjectKey = Session.Project.useSelectOptionalSelected();
  const { data: activeProject } = PProject.useRetrieve(
    { key: activeProjectKey ?? "" },
    { beforeRetrieve: ({ query: { key } }) => primitive.isNonZero(key) },
  );
  return (
    <ComponentNav.Bar
      location="top"
      size="6rem"
      data-tauri-drag-region
      bordered={false}
      className={CSS.BE("mosaic", "bar")}
    >
      <Nav.Bar.Start data-tauri-drag-region align="center">
        <Window.Controls visibleIfOS="macOS" forceOS={os} />
        {os === "Windows" && (
          <>
            <Logo />
            <BottomToggleButton />
          </>
        )}
      </Nav.Bar.Start>
      <Nav.Bar.AbsoluteCenter>
        <Text.Text
          level="small"
          weight={500}
          color={10}
          data-tauri-drag-region
          style={{ cursor: "default" }}
        >
          {activeName} {activeProject?.name && `- ${activeProject.name}`}
        </Text.Text>
      </Nav.Bar.AbsoluteCenter>
      <Nav.Bar.End data-tauri-drag-region align="center" justify="end">
        <Window.Controls visibleIfOS="Windows" forceOS={os} />
        {os === "macOS" && <BottomToggleButton />}
      </Nav.Bar.End>
    </ComponentNav.Bar>
  );
};
