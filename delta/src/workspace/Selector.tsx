// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, type ReactElement } from "react";

import { Select, Synnax } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { useSelectActive, useSelectActiveKey } from "@/workspace/selectors";
import { setActive } from "@/workspace/slice";

export const Selector = (): ReactElement => {
  const client = Synnax.use();
  const d = useDispatch();
  const active = useSelectActive();
  const handleChange = useCallback((v: string) => d(setActive(v)), [d]);
  return (
    <Select.Single
      allowClear={false}
      searcher={client?.workspaces}
      onChange={handleChange}
      value={active?.key}
      columns={[
        {
          key: "name",
          name: "Name",
        },
      ]}
      tagKey="name"
    />
  );
};
