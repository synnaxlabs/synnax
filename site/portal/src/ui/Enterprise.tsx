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

import { Panel } from "@/ui/Panel";

export const CONTACT_URL = "https://synnaxlabs.com/#contact";

/** Enterprise tells a personal user what the enterprise edition adds and how to ask. */
export const Enterprise = (): ReactElement => (
  <Panel
    x
    justify="between"
    align="center"
    gap="large"
    wrap
    style={{ padding: "3.5rem 4rem" }}
  >
    <Flex.Box y gap="tiny" style={{ flex: "1 1 40rem" }}>
      <Text.Text level="h5" weight={500} color={11}>
        Synnax Enterprise
      </Text.Text>
      <Text.Text level="p" color={9}>
        A standalone Core on your own hardware, licensed machines, and a team that
        shares licenses. Organizations are set up by Synnax Labs.
      </Text.Text>
    </Flex.Box>
    <Button.Button variant="outlined" href={CONTACT_URL}>
      <Icon.Feedback />
      Talk to us
    </Button.Button>
  </Panel>
);
