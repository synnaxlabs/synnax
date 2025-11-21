// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Nav, Synnax, Text } from "@synnaxlabs/pluto";

import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export const POLICY_CREATE_LAYOUT_TYPE = "createPolicy";

export const POLICY_CREATE_LAYOUT: Layout.BaseState = {
  key: POLICY_CREATE_LAYOUT_TYPE,
  type: POLICY_CREATE_LAYOUT_TYPE,
  icon: "Policy",
  location: "modal",
  name: "Policy.Create",
  window: {
    resizable: false,
    size: { height: 400, width: 650 },
    navTop: true,
  },
};

export const Create: Layout.Renderer = ({ onClose }) => {
  const client = Synnax.use();

  return (
    <Flex.Box grow empty>
      <Flex.Box
        className="console-form"
        justify="center"
        align="center"
        style={{ padding: "1rem 3rem" }}
        grow
      >
        <Text.Text level="h3" shade={6}>
          Policy creation form coming soon...
        </Text.Text>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText action="Create" />
        <Nav.Bar.End style={{ paddingRight: "2rem" }}>
          <Button.Button
            onClick={() => onClose()}
            disabled={client == null}
            tooltip={
              client == null
                ? "No Core Connected"
                : `Save to ${client.params.name ?? "Synnax"}`
            }
            tooltipLocation="bottom"
            trigger={Triggers.SAVE}
            variant="filled"
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
