// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Input } from "@synnaxlabs/pluto";
import { memo, type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { useSelectControlAuthority } from "@/schematic/selectors";
import { setControlAuthority } from "@/schematic/slice";

export interface ControlControlsProps {
  layoutKey: string;
}

export const ControlControls = memo(
  ({ layoutKey }: ControlControlsProps): ReactElement => {
    const controlAuthority = useSelectControlAuthority(layoutKey);
    const dispatch = useDispatch();
    const handleChange = (authority: number): void => {
      dispatch(setControlAuthority({ key: layoutKey, authority }));
    };
    return (
      <Align.Space style={{ height: "100%", padding: "1.5rem" }} direction="y">
        <Input.Item label="Default Control Authority" align="start">
          <Input.Numeric
            value={controlAuthority}
            onChange={handleChange}
            min={0}
            max={255}
          />
        </Input.Item>
      </Align.Space>
    );
  },
);
ControlControls.displayName = "ControlControls";
