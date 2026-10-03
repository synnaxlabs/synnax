// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { type ReactElement } from "react";

/** Panel is the raised surface every block of portal content sits on. */
export const Panel = (props: Flex.BoxProps): ReactElement => (
  <Flex.Box
    y
    bordered
    borderColor={4}
    rounded="large"
    background={2}
    full="x"
    {...props}
  />
);
