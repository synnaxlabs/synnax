// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Input } from "@synnaxlabs/pluto";
import { control } from "@synnaxlabs/x";

import { CSS } from "@/css";
import {
  useSelectAuthority,
  useSelectLegendVisible,
  useSetControlAuthority,
  useSetLegendVisible,
} from "@/schematic/session/slice";

export const Control = () => {
  const authority = useSelectAuthority();
  const handleAuthorityChange = useSetControlAuthority();
  const legendVisible = useSelectLegendVisible();
  const handleVisibleChange = useSetLegendVisible();
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
        <Input.Switch value={legendVisible} onChange={handleVisibleChange} />
      </Input.Item>
    </Flex.Box>
  );
};
