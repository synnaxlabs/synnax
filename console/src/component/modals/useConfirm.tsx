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

import { Body } from "@/component/modals/Body";
import { createPrompt, type Prompt } from "@/component/modals/factory";
import { Footer } from "@/component/modals/Footer";
import { Frame } from "@/component/modals/Frame";
import { Header } from "@/component/modals/Header";
import { type ContentProps } from "@/session/modals/Context";
import { Triggers } from "@/component/triggers";

interface ButtonProps {
  variant?: status.Variant;
  label?: string;
  delay?: number;
}

export interface ConfirmParams {
  message: string;
  description: string;
  confirm?: ButtonProps;
  cancel?: ButtonProps;
  title?: string;
  icon?: Icon.ReactElement;
}

export interface PromptConfirm extends Prompt<boolean, ConfirmParams> {}

interface InternalButtonProps
  extends ButtonProps, Omit<Button.ButtonProps, "variant"> {}

const DEFAULT_CONFIRM_LABEL = "Confirm";

const ConfirmButton = ({
  label = DEFAULT_CONFIRM_LABEL,
  variant = "error",
  delay,
  ...rest
}: InternalButtonProps) => (
  <Button.Button
    variant="filled"
    status={variant}
    trigger={Triggers.SAVE}
    onClickDelay={delay}
    {...rest}
  >
    {label}
  </Button.Button>
);

const CancelButton = ({
  label = "Cancel",
  variant,
  delay,
  ...rest
}: InternalButtonProps) => (
  <Button.Button
    variant="outlined"
    status={variant}
    trigger={Triggers.SAVE}
    onClickDelay={delay}
    {...rest}
  >
    {label}
  </Button.Button>
);
const Confirm = ({
  message,
  description,
  confirm = {},
  cancel = {},
  title = "Confirm",
  icon,
  close,
}: ContentProps<ConfirmParams, boolean>) => (
  <Frame>
    <Header icon={icon}>{title}</Header>
    <Body>
      <Text.Text level="h3" weight={450}>
        {message}
      </Text.Text>
      <Text.Text weight={450}>{description}</Text.Text>
    </Body>
    <Footer>
      <Triggers.SaveHelpText action={confirm.label ?? DEFAULT_CONFIRM_LABEL} />
      <Nav.Bar.End x align="center">
        <CancelButton {...cancel} onClick={() => close(false)} />
        <ConfirmButton {...confirm} onClick={() => close(true)} />
      </Nav.Bar.End>
    </Footer>
  </Frame>
);

export const useConfirm = createPrompt<boolean, ConfirmParams>(Confirm);
