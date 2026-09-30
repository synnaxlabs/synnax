// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/modals/useConfirm.css";

import { type status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";

import { CSS } from "@/platform/css";
import { createPrompt, type Prompt } from "@/platform/modals/factory";
import { Triggers } from "@/platform/triggers";
import { type Session } from "@/session";

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
    onClickDelay={delay}
    triggerIndicator={PTriggers.ESCAPE}
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
}: Session.Modals.ContentProps<ConfirmParams, boolean>) => (
  <Modal.Frame className={CSS.B("confirm")}>
    <Modal.Header icon={icon}>{title}</Modal.Header>
    <Modal.Body>
      <Text.Text level="h3" weight={450}>
        {message}
      </Text.Text>
      <Text.Text weight={450}>{description}</Text.Text>
    </Modal.Body>
    <Modal.Footer>
      <Triggers.SaveHelpText action={confirm.label ?? DEFAULT_CONFIRM_LABEL} />
      <Nav.Bar.End x align="center">
        <CancelButton {...cancel} onClick={() => close(false)} />
        <ConfirmButton {...confirm} onClick={() => close(true)} />
      </Nav.Bar.End>
    </Modal.Footer>
  </Modal.Frame>
);

export const useConfirm = createPrompt<boolean, ConfirmParams>(Confirm);
