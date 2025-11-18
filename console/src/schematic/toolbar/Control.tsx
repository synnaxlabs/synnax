// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Input } from "@synnaxlabs/pluto";
import { control } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

export const Control = () => {
  const dispatch = useDispatch();
  const authority = useSelectAuthority(layoutKey);

  return (
    <Flex.Box x gap="small" style={{ padding: "1.5rem 2rem" }}>
      <Input.Item label="Control Authority">
        <Input.Numeric
          value={authority ?? 0}
          onChange={(v) => dispatch(setAuthority({ key: layoutKey, authority: v }))}
          bounds={control.AUTHORITY_BOUNDS}
        />
      </Input.Item>
    </Flex.Box>
  );
};
