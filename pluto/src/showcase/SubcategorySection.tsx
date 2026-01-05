// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@/flex";
import { Text } from "@/text";

export const SubcategorySection = ({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) => (
  <Flex.Box y gap="huge" style={{ padding: "5rem" }} grow bordered>
    <Flex.Box y gap="medium">
      <Text.Text level="h4">{title}</Text.Text>
      <Text.Text level="small" color={9}>
        {description}
      </Text.Text>
    </Flex.Box>
    <Flex.Box>{children}</Flex.Box>
  </Flex.Box>
);
