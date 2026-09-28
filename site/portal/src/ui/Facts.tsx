// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { type PropsWithChildren, type ReactElement } from "react";

/** Facts is a panel of labeled values laid out on a grid. */
export const Facts = ({ children }: PropsWithChildren): ReactElement => (
  <Flex.Box bordered rounded background={1} style={{ padding: "3rem 4rem" }}>
    <Flex.Box className="portal-facts" full="x">
      {children}
    </Flex.Box>
  </Flex.Box>
);

export interface FactProps {
  label: string;
  value: string;
  /** code shows the value in a monospace face, for keys and hashes. */
  code?: boolean;
}

/** Fact is one labeled value in {@link Facts}. */
export const Fact = ({ label, value, code = false }: FactProps): ReactElement => (
  <Flex.Box y gap="tiny" style={{ minWidth: 0 }}>
    <Text.Text level="small" color={9}>
      {label}
    </Text.Text>
    <Text.Text level="p" variant={code ? "code" : "prose"} overflow="ellipsis">
      {value}
    </Text.Text>
  </Flex.Box>
);
