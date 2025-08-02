// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Nav, Text, Triggers } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

export const SAVE: Triggers.Trigger = ["Control", "Enter"];

export interface SaveHelpTextProps {
  action?: string;
  noBar?: boolean;
  trigger?: Triggers.Trigger;
}

export const SaveHelpText = ({
  action = "Save",
  noBar = false,
  trigger = SAVE,
}: SaveHelpTextProps): ReactElement => {
  const content = (
    <>
      <Flex.Box x empty>
        <Triggers.Text shade={11} level="small" trigger={trigger} />
      </Flex.Box>
      <Text.Text shade={11} level="small">
        {action}
      </Text.Text>
    </>
  );
  return noBar ? content : <Nav.Bar.Start gap="small">{content}</Nav.Bar.Start>;
};
