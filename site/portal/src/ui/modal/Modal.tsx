// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Form as BaseForm } from "@synnaxlabs/lyra/form";
import { Modal as Base } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";
import { type z } from "zod";

import { Submit } from "@/ui/Submit";
import { useAction } from "@/ui/useAction";

export interface FrameProps
  extends PropsWithChildren, Pick<Dialog.FrameProps, "visible" | "onVisibleChange"> {
  /** trigger opens the modal. Leave it out to control visibility from outside. */
  trigger?: ReactNode;
  /** name is the modal's title. Dots split it into breadcrumb segments. */
  name: string;
  icon?: Base.HeaderProps["icon"];
  className?: string;
}

/**
 * Frame is a portal modal: the Lyra modal anatomy under the portal's density scope,
 * opened by `trigger`. Children render inside the dialog and read
 * `Dialog.useContext().close` to dismiss it.
 */
export const Frame = ({
  trigger,
  visible,
  onVisibleChange,
  name,
  icon,
  className,
  children,
}: FrameProps): ReactElement => (
  <Dialog.Frame variant="modal" visible={visible} onVisibleChange={onVisibleChange}>
    {trigger}
    <Base.Frame className={`portal-modal ${className ?? ""}`}>
      <Base.Header icon={icon}>{name}</Base.Header>
      {children}
    </Base.Frame>
  </Dialog.Frame>
);

export const Body = Base.Body;

export interface FooterProps {
  /** error reads at the start of the footer when set. */
  error?: string | null;
  /** hint reads at the start of the footer when there is no error, e.g. to say an
   * action needs a press and hold. */
  hint?: string;
  children: ReactNode;
}

/** Footer is the modal's action bar. Put the primary action last. */
export const Footer = ({ error, hint, children }: FooterProps): ReactElement => (
  <Base.Footer>
    {error != null ? (
      <Nav.Bar.Start>
        <Status.Summary variant="error" level="small" message={error} />
      </Nav.Bar.Start>
    ) : (
      hint != null && (
        <Nav.Bar.Start>
          <Text.Text level="small" color={9}>
            {hint}
          </Text.Text>
        </Nav.Bar.Start>
      )
    )}
    <Nav.Bar.End x align="center" gap="small">
      {children}
    </Nav.Bar.End>
  </Base.Footer>
);

/** Cancel closes the enclosing modal. */
export const Cancel = ({ children = "Cancel" }: PropsWithChildren): ReactElement => {
  const { close } = Dialog.useContext();
  return (
    <Button.Button
      variant="outlined"
      onClick={close}
      triggerIndicator={Triggers.ESCAPE}
    >
      {children}
    </Button.Button>
  );
};

export interface FormProps<Z extends z.ZodType> extends PropsWithChildren {
  methods: BaseForm.UseReturn<Z>;
  /** submit is the content of the primary button. */
  submit: ReactNode;
  /**
   * onSubmit runs with the form's value once it validates. The modal closes when it
   * resolves and shows its error when it rejects.
   */
  onSubmit: (value: z.infer<Z>) => Promise<void>;
}

/** Form is a modal body of fields over a footer that submits them. */
export const Form = <Z extends z.ZodType>({
  methods,
  submit,
  onSubmit,
  children,
}: FormProps<Z>): ReactElement => {
  const { close } = Dialog.useContext();
  const action = useAction(async () => {
    if (!methods.validate()) return;
    await onSubmit(methods.value());
    close();
  });
  return (
    <BaseForm.Form<Z> {...methods}>
      <Body gap="medium">{children}</Body>
      <Footer error={action.error}>
        <Cancel />
        <Submit action={action}>{submit}</Submit>
      </Footer>
    </BaseForm.Form>
  );
};

export interface ConfirmProps extends PropsWithChildren {
  /** question heads the modal, like "Revoke this license?". */
  question: ReactNode;
  /** confirm is the label of the button that must be held. */
  confirm: string;
  /** delay is how long the button must be held, in milliseconds. */
  delay?: number;
  /** onConfirm runs once the hold completes. The modal closes when it resolves. */
  onConfirm: () => Promise<void>;
}

/**
 * Confirm asks before a destructive action. Children explain what happens, and the
 * user holds the confirm button to go ahead.
 */
export const Confirm = ({
  question,
  confirm,
  delay = 1000,
  onConfirm,
  children,
}: ConfirmProps): ReactElement => {
  const { close } = Dialog.useContext();
  const action = useAction(async () => {
    await onConfirm();
    close();
  });
  return (
    <>
      <Body gap="small">
        <Text.Text level="h4" weight={450}>
          {question}
        </Text.Text>
        <Text.Text level="p" color={9}>
          {children}
        </Text.Text>
      </Body>
      <Footer error={action.error} hint={`Press and hold ${confirm} to confirm`}>
        <Cancel />
        <Button.Button
          variant="filled"
          status={action.loading ? "loading" : "error"}
          onClick={action.run}
          onClickDelay={delay}
        >
          {confirm}
        </Button.Button>
      </Footer>
    </>
  );
};
