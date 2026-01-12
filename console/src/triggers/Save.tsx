// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Nav, Triggers } from "@synnaxlabs/pluto";
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
    <Triggers.Text level="small" trigger={trigger}>
      {action}
    </Triggers.Text>
  );
  return noBar ? content : <Nav.Bar.Start gap="small">{content}</Nav.Bar.Start>;
};
