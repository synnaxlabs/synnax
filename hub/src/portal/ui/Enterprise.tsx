// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Icon, Text } from "@synnaxlabs/pluto";
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
    style={{ padding: "3rem 4rem" }}
  >
    <Flex.Box y gap="small" style={{ flex: "1 1 40rem" }}>
      <Text.Text level="h5">Synnax Enterprise</Text.Text>
      <Text.Text level="p" color={10}>
        A standalone Core on your own hardware, licensed machines, and a team that
        shares licenses and support. Organizations are set up by Synnax Labs.
      </Text.Text>
    </Flex.Box>
    <Button.Button variant="filled" href={CONTACT_URL}>
      <Icon.Feedback />
      Talk to us
    </Button.Button>
  </Flex.Box>
);
