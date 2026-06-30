// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Input, Schematic } from "@synnaxlabs/pluto";
import { control } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/component/css";
import { Session } from "@/session";

export const Control = () => {
  const dispatch = useDispatch();
  const authority = Session.Schematic.useSelectAuthority();
  const legendVisible = Session.Schematic.useSelectLegendVisible();
  const key = Schematic.useKey();
  const handleAuthorityChange = useCallback(
    (authority: control.Authority) => {
      dispatch(Session.Schematic.setControlAuthority({ key, authority }));
    },
    [key],
  );
  const handleLegendVisibleChange = useCallback(
    (visible: boolean) => {
      dispatch(Session.Schematic.setLegendVisible({ key, visible }));
    },
    [key],
  );
  return (
    <Flex.Box x gap="small" className={CSS.BE("schematic", "control")}>
      <Input.Item label="Control authority">
        <Input.Numeric
          value={authority}
          onChange={handleAuthorityChange}
          bounds={control.AUTHORITY_BOUNDS}
        />
      </Input.Item>
      <Input.Item label="Show control state legend">
        <Input.Switch value={legendVisible} onChange={handleLegendVisibleChange} />
      </Input.Item>
    </Flex.Box>
  );
};
