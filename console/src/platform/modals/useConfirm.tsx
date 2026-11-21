// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/modals/useConfirm.css";

import type { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import type { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";
import { type PropsWithChildren, useId, useState } from "react";

import { CSS } from "@/platform/css";
import { createPrompt } from "@/platform/modals/factory";
import { Triggers } from "@/platform/triggers";
import type { Session } from "@/session";

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
    triggerIndicator
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
interface LayoutProps extends ConfirmParams, PropsWithChildren {
  onConfirm: () => void;
  onCancel: () => void;
}

const Layout = ({
  message,
  description,
  confirm = {},
  cancel = {},
  title = "Confirm",
  icon,
  onConfirm,
  onCancel,
  children,
}: LayoutProps) => (
  <Modal.Frame className={CSS.B("confirm")}>
    <Modal.Header icon={icon}>{title}</Modal.Header>
    <Modal.Body gap="medium" className={CSS.BE("confirm", "body")}>
      <Text.Text level="h3" weight={450}>
        {message}
      </Text.Text>
      <Text.Text weight={450}>{description}</Text.Text>
      {children}
    </Modal.Body>
    <Modal.Footer>
      <Nav.Bar.End x align="center">
        <CancelButton {...cancel} onClick={onCancel} />
        <ConfirmButton {...confirm} onClick={onConfirm} />
      </Nav.Bar.End>
    </Modal.Footer>
  </Modal.Frame>
);

const Confirm = ({
  close,
  ...rest
}: Session.Modals.ContentProps<ConfirmParams, boolean>) => (
  <Layout {...rest} onConfirm={() => close(true)} onCancel={() => close(false)} />
);

export const useConfirm = createPrompt<boolean, ConfirmParams>(Confirm);

/** A checkbox in a confirm. It is off when the confirm opens. */
export interface ConfirmOption {
  label: string;
  /** Shows under the label. */
  description?: string;
}

interface ConfirmWithOptionParams extends ConfirmParams {
  option: ConfirmOption;
}

interface OptionAnswer {
  option: boolean;
}

const ConfirmWithOption = ({
  option: { label, description },
  close,
  ...rest
}: Session.Modals.ContentProps<ConfirmWithOptionParams, OptionAnswer>) => {
  const id = useId();
  const labelID = `${id}-label`;
  const [checked, setChecked] = useState(false);
  return (
    <Layout
      {...rest}
      onConfirm={() => close({ option: checked })}
      onCancel={() => close()}
    >
      <Flex.Box x gap="small" align="start">
        <Input.Checkbox
          id={id}
          aria-labelledby={labelID}
          value={checked}
          onChange={setChecked}
        />
        <Flex.Box y>
          {/* As tall as the checkbox, so the label stays level with it. */}
          <Flex.Box x align="center" size="medium">
            <Text.Text el="label" id={labelID} htmlFor={id} weight={450}>
              {label}
            </Text.Text>
          </Flex.Box>
          {description != null && <Text.Text color={9}>{description}</Text.Text>}
        </Flex.Box>
      </Flex.Box>
    </Layout>
  );
};

/**
 * Opens a confirm with a checkbox. Resolves with the state of the checkbox, or null
 * when the user cancels.
 */
export const useConfirmWithOption = createPrompt<OptionAnswer, ConfirmWithOptionParams>(
  ConfirmWithOption,
);
