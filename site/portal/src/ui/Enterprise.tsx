// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

export const CONTACT_URL = "https://synnaxlabs.com/#contact";

/** Enterprise tells a personal user what the enterprise edition adds and how to ask. */
export const Enterprise = (): ReactElement => (
  <Flex.Box
    x
    justify="between"
    align="center"
    gap="large"
    wrap
    bordered
    rounded
    background={1}
    className="portal-enterprise"
  >
    <Flex.Box y gap="small" className="portal-enterprise__text">
      <Text.Text level="h5">Synnax Enterprise</Text.Text>
      <Text.Text level="p" color={10}>
        A standalone Core on your own hardware, licensed machines, and a team that
        shares licenses. Organizations are set up by Synnax Labs.
      </Text.Text>
    </Flex.Box>
    <Button.Button variant="filled" href={CONTACT_URL}>
      <Icon.Feedback />
      Talk to us
    </Button.Button>
  </Flex.Box>
);
