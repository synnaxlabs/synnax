// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Form } from "@synnaxlabs/lyra/form";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useState } from "react";
import { z } from "zod";

import { Card } from "@/ui/auth/Card";
import { withTarget } from "@/ui/auth/redirect";
import { start } from "@/ui/auth/session";
import { useClerk } from "@/ui/clerk";
import { useAction } from "@/ui/useAction";

const emailSchema = z.object({ email: z.email("Enter your email address") });

const resetSchema = z.object({
  code: z.string().trim().min(1, "Enter the code"),
  password: z.string().min(8, "Use at least 8 characters"),
});

type Step = "email" | "reset";

export interface ResetProps {
  /** target is where to land after setting the new password. */
  target: string;
}

/** Reset lets a user set a new password with a code sent to their email. */
export const Reset = ({ target }: ResetProps): ReactElement => {
  const clerk = useClerk();
  const [step, setStep] = useState<Step>("email");
  const emailMethods = Form.use({ values: { email: "" }, schema: emailSchema });
  const resetMethods = Form.use({
    values: { code: "", password: "" },
    schema: resetSchema,
  });

  const send = useAction(async () => {
    if (clerk?.client == null || !emailMethods.validate()) return;
    const res = await clerk.client.signIn.create({
      identifier: emailMethods.value().email,
    });
    const factor = res.supportedFirstFactors?.find(
      (f) => f.strategy === "reset_password_email_code",
    );
    if (factor?.strategy !== "reset_password_email_code")
      throw new Error("This account cannot reset its password by email");
    await res.prepareFirstFactor({
      strategy: "reset_password_email_code",
      emailAddressId: factor.emailAddressId,
    });
    setStep("reset");
  });

  const reset = useAction(async () => {
    if (clerk?.client == null || !resetMethods.validate()) return;
    const { code, password } = resetMethods.value();
    const res = await clerk.client.signIn.attemptFirstFactor({
      strategy: "reset_password_email_code",
      code,
      password,
    });
    if (res.status !== "complete") throw new Error("That code did not work");
    await start(clerk, res.createdSessionId, target);
  });

  const footer = (
    <Text.Text
      el="a"
      level="small"
      variant="link"
      href={withTarget("/sign-in", target)}
    >
      Back to sign in
    </Text.Text>
  );

  if (step === "reset")
    return (
      <Card
        title="Set a new password"
        description={`We sent a code to ${emailMethods.value().email}.`}
        error={reset.error}
        footer={footer}
      >
        <Form.Form<typeof resetSchema> {...resetMethods}>
          <Form.TextField
            path="code"
            label="Code"
            inputProps={{ autoFocus: true, autoComplete: "one-time-code" }}
          />
          <Form.TextField
            path="password"
            label="New password"
            inputProps={{ type: "password", autoComplete: "new-password" }}
          />
          <Button.Button
            variant="filled"
            size="large"
            full="x"
            justify="center"
            onClick={reset.run}
            status={reset.loading ? "loading" : undefined}
            trigger={["Enter"]}
          >
            Set password and sign in
          </Button.Button>
        </Form.Form>
      </Card>
    );

  return (
    <Card
      title="Reset your password"
      description="We will email you a code."
      error={send.error}
      footer={footer}
    >
      <Form.Form<typeof emailSchema> {...emailMethods}>
        <Form.TextField
          path="email"
          label="Email"
          inputProps={{ type: "email", autoComplete: "email", autoFocus: true }}
        />
        <Button.Button
          variant="filled"
          size="large"
          full="x"
          justify="center"
          onClick={send.run}
          disabled={clerk == null}
          status={send.loading ? "loading" : undefined}
          trigger={["Enter"]}
        >
          Send code
        </Button.Button>
      </Form.Form>
    </Card>
  );
};
