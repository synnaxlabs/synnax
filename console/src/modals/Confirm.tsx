// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Button, Nav, type Status, Text } from "@synnaxlabs/pluto";

import { Layout } from "@/layout";
import { type BaseArgs, createBase, type Prompt } from "@/modals/Base";
import { Triggers } from "@/triggers";

export interface PromptConfirmLayoutArgs extends BaseArgs<boolean> {
  message: string;
  description: string;
  confirm?: { variant?: Status.Variant; label?: string };
  cancel?: { variant?: Status.Variant; label?: string };
}

export const CONFIRM_LAYOUT_TYPE = "confirm";

export interface PromptConfirm extends Prompt<boolean, PromptConfirmLayoutArgs> {}

export const [useConfirm, Confirm] = createBase<boolean, PromptConfirmLayoutArgs>(
  "Confirm",
  CONFIRM_LAYOUT_TYPE,
  ({ value: { message, description, confirm, cancel }, onFinish }) => {
    const { variant: confirmVariant = "error", label: confirmLabel = "Confirm" } =
      confirm ?? {};
    const { variant: cancelVariant, label: cancelLabel = "Cancel" } = cancel ?? {};
    return (
      <Align.Space direction="y" grow justify="center">
        <Align.Space
          direction="y"
          grow
          align="start"
          justify="center"
          style={{ padding: "5rem" }}
        >
          <Text.Text level="h3" shade={9} weight={450}>
            {message}
          </Text.Text>
          <Text.Text level="p" shade={7} weight={450}>
            {description}
          </Text.Text>
        </Align.Space>
        <Layout.BottomNavBar>
          <Triggers.SaveHelpText action={confirmLabel} />
          <Nav.Bar.End direction="x" align="center">
            <Button.Button
              variant="outlined"
              status={cancelVariant}
              onClick={() => onFinish(false)}
            >
              {cancelLabel}
            </Button.Button>
            <Button.Button
              status={confirmVariant}
              onClick={() => onFinish(true)}
              triggers={Triggers.SAVE}
            >
              {confirmLabel}
            </Button.Button>
          </Nav.Bar.End>
        </Layout.BottomNavBar>
      </Align.Space>
    );
  },
);
