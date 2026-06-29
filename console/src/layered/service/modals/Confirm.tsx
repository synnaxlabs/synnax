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

import { Body } from "@/layered/service/modals/Body";
import { createPrompt, type Prompt } from "@/layered/service/modals/factory";
import { type ContentProps } from "@/layered/session/modals/store";
import { Triggers } from "@/triggers";

import { Footer } from "./Footer";
import { Frame } from "./Frame";
import { Header } from "./Header";

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
  icon?: Icon.ReactElement;
}

export interface PromptConfirm extends Prompt<boolean, ConfirmParams> {}

const Confirm = ({
  message,
  description,
  confirm = {},
  cancel = {},
  title = "Confirm",
  icon,
  close,
}: ContentProps<ConfirmParams, boolean>) => {
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
  return (
    <Frame>
      <Header icon={icon}>{title}</Header>
      <Body>
        <Text.Text level="h3" weight={450}>
          {message}
        </Text.Text>
        <Text.Text weight={450}>{description}</Text.Text>
      </Body>
      <Footer>
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
      </Footer>
    </Frame>
  );
};

export const useConfirm = createPrompt<boolean, ConfirmParams>(Confirm);
