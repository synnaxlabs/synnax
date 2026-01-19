// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Nav, Text } from "@synnaxlabs/pluto";
import { type status } from "@synnaxlabs/x";

import { type BaseArgs, createBase, type Prompt } from "@/modals/Base";
import { ModalContentLayout } from "@/modals/layout";
import { Triggers } from "@/triggers";

interface ConfirmButtonProps {
  variant?: status.Variant;
  label?: string;
  delay?: number;
}

export interface PromptConfirmLayoutArgs extends BaseArgs<boolean> {
  message: string;
  description: string;
  confirm?: ConfirmButtonProps;
  cancel?: ConfirmButtonProps;
}

export const CONFIRM_LAYOUT_TYPE = "confirm";

export interface PromptConfirm extends Prompt<boolean, PromptConfirmLayoutArgs> {}

export const [useConfirm, Confirm] = createBase<boolean, PromptConfirmLayoutArgs>(
  "Confirm",
  CONFIRM_LAYOUT_TYPE,
  ({ value: { message, description, confirm = {}, cancel = {} }, onFinish }) => {
    const {
      variant: confirmVariant = "error",
      label: confirmLabel = "Confirm",
      delay: confirmDelay = 0,
    } = confirm;
    const {
      variant: cancelVariant,
      label: cancelLabel = "Cancel",
      delay: cancelDelay = 0,
    } = cancel;
    const footer = (
      <>
        <Triggers.SaveHelpText action={confirmLabel} />
        <Nav.Bar.End x align="center">
          <Button.Button
            status={cancelVariant}
            onClick={() => onFinish(false)}
            onClickDelay={cancelDelay}
          >
            {cancelLabel}
          </Button.Button>
          <Button.Button
            variant="filled"
            status={confirmVariant}
            onClick={() => onFinish(true)}
            trigger={Triggers.SAVE}
            onClickDelay={confirmDelay}
          >
            {confirmLabel}
          </Button.Button>
        </Nav.Bar.End>
      </>
    );

    return (
      <ModalContentLayout footer={footer}>
        <Text.Text level="h3" weight={450}>
          {message}
        </Text.Text>
        <Text.Text weight={450}>{description}</Text.Text>
      </ModalContentLayout>
    );
  },
);
