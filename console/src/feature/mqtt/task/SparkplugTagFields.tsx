// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";

import { type SparkplugTagID } from "@/feature/mqtt/task/types";

export interface SparkplugNodeFieldsProps {
  path: deep.Target<Pick<SparkplugTagID, "group" | "edgeNode">>;
}

export const SparkplugNodeFields = ({ path }: SparkplugNodeFieldsProps) => {
  const node = deep.widen(path);
  return (
    <Flex.Box x>
      <PForm.TextField
        path={node.group}
        label="Group"
        inputProps={GROUP_INPUT_PROPS}
        grow
      />
      <PForm.TextField
        path={node.edgeNode}
        label="Edge node"
        inputProps={EDGE_NODE_INPUT_PROPS}
        grow
      />
    </Flex.Box>
  );
};

export interface SparkplugTagFieldsProps {
  path: deep.Target<SparkplugTagID>;
}

export const SparkplugTagFields = ({ path }: SparkplugTagFieldsProps) => {
  const tag = deep.widen(path);
  return (
    <>
      <SparkplugNodeFields path={path} />
      <Flex.Box x>
        <PForm.TextField
          path={tag.device}
          label="Device"
          inputProps={DEVICE_INPUT_PROPS}
          grow
        />
        <PForm.TextField path={tag.tag} label="Tag" inputProps={TAG_INPUT_PROPS} grow />
      </Flex.Box>
    </>
  );
};

const GROUP_INPUT_PROPS = { placeholder: "plant" } as const;

const EDGE_NODE_INPUT_PROPS = { placeholder: "line1" } as const;

const DEVICE_INPUT_PROPS = { placeholder: "Optional" } as const;

const TAG_INPUT_PROPS = { placeholder: "oven/temperature" } as const;
