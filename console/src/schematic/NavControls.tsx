// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Viewport } from "@synnaxlabs/pluto";
import { type ReactElement, useMemo } from "react";
import { useDispatch } from "react-redux";

import { useSelectViewportMode } from "@/schematic/selectors";
import { setViewportMode } from "@/schematic/slice";

export const NavControls = (): ReactElement => {
  const mode = useSelectViewportMode();
  const d = useDispatch();

  const handleModeChange = (mode: Viewport.Mode): void => {
    d(setViewportMode({ mode }));
  };

  const triggers = useMemo(() => Viewport.DEFAULT_TRIGGERS[mode], [mode]);

  return (
    <Viewport.SelectMode
      bordered={false}
      rounded={false}
      value={mode}
      onChange={handleModeChange}
      triggers={triggers}
      disable={["zoom", "click", "zoomReset"]}
    />
  );
};
