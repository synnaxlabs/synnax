// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Input, Schematic as Base } from "@synnaxlabs/pluto";
import { control } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { useSelectLegendVisible } from "@/schematic/selectors";
import { setLegendVisible } from "@/schematic/slice";

export const Control = ({ layoutKey }: { layoutKey: string }) => {
  const dispatch = useDispatch();
  const authority = Base.useSelectAuthority({ key: layoutKey });
  const legendVisible = useSelectLegendVisible(layoutKey);

  return (
    <Flex.Box x gap="small" style={{ padding: "1.5rem 2rem" }}>
      <Input.Item label="Control Authority">
        <Input.Numeric
          value={authority ?? 0}
          onChange={() => {}}
          bounds={control.AUTHORITY_BOUNDS}
        />
      </Input.Item>
      <Input.Item label="Show Control State Legend">
        <Input.Switch
          value={legendVisible ?? true}
          onChange={(v) => dispatch(setLegendVisible({ key: layoutKey, visible: v }))}
        />
      </Input.Item>
    </Flex.Box>
  );
};
