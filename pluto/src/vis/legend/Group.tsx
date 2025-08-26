// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Flex } from "@/flex";
import { Text } from "@/text";
import { Entries, type EntriesProps } from "@/vis/legend/Entries";

export interface GroupProps extends EntriesProps {
  name: string;
}

export const Group = ({ name, ...rest }: GroupProps): ReactElement => (
  <Flex.Box x>
    <Text.Text size="small">{name}</Text.Text>
    <Flex.Box y>
      <Entries {...rest} />
    </Flex.Box>
  </Flex.Box>
);
