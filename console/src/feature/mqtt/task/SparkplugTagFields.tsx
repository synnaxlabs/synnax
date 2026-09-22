// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Form as PForm } from "@synnaxlabs/pluto";

export interface SparkplugTagFieldsProps {
  path: string;
}

export const SparkplugTagFields = ({ path }: SparkplugTagFieldsProps) => (
  <>
    <Flex.Box x>
      <PForm.TextField
        path={`${path}.group`}
        label="Group"
        inputProps={GROUP_INPUT_PROPS}
        grow
      />
      <PForm.TextField
        path={`${path}.edgeNode`}
        label="Edge node"
        inputProps={EDGE_NODE_INPUT_PROPS}
        grow
      />
    </Flex.Box>
    <Flex.Box x>
      <PForm.TextField
        path={`${path}.device`}
        label="Device"
        inputProps={DEVICE_INPUT_PROPS}
        grow
      />
      <PForm.TextField
        path={`${path}.tag`}
        label="Tag"
        inputProps={TAG_INPUT_PROPS}
        grow
      />
    </Flex.Box>
  </>
);

const GROUP_INPUT_PROPS = { placeholder: "plant" } as const;

const EDGE_NODE_INPUT_PROPS = { placeholder: "line1" } as const;

const DEVICE_INPUT_PROPS = { placeholder: "Optional" } as const;

const TAG_INPUT_PROPS = { placeholder: "oven/temperature" } as const;
