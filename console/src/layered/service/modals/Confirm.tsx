// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Button, type Icon, Nav, Text } from "@synnaxlabs/pluto";

import { type Prompt, prompt } from "@/layered/service/modals/Base";
import { ModalContentLayout } from "@/layered/service/modals/layout";
import { Triggers } from "@/triggers";

interface ConfirmButtonProps {
  variant?: status.Variant;
  label?: string;
  delay?: number;
}

export interface ConfirmParams {
  message: string;
  description: string;
  confirm?: ConfirmButtonProps;
  cancel?: ConfirmButtonProps;
  title?: string;
  icon?: Icon.ReactElement | string;
}

export interface PromptConfirm extends Prompt<boolean, ConfirmParams> {}

export const useConfirm = prompt<boolean, ConfirmParams>(
  ({
    params: {
      message,
      description,
      confirm = {},
      cancel = {},
      title = "Confirm",
      icon,
    },
    close,
  }) => {
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
            onClick={() => close(false)}
            onClickDelay={cancelDelay}
          >
            {cancelLabel}
          </Button.Button>
          <Button.Button
            variant="filled"
            status={confirmVariant}
            onClick={() => close(true)}
            trigger={Triggers.SAVE}
            onClickDelay={confirmDelay}
          >
            {confirmLabel}
          </Button.Button>
        </Nav.Bar.End>
      </>
    );

    return (
      <ModalContentLayout title={title} icon={icon} footer={footer}>
        <Text.Text level="h3" weight={450}>
          {message}
        </Text.Text>
        <Text.Text weight={450}>{description}</Text.Text>
      </ModalContentLayout>
    );
  },
);
