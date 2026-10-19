// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useState } from "react";
import { z } from "zod";

import { Card } from "@/ui/auth/Card";
import { OAuth } from "@/ui/auth/OAuth";
import { withTarget } from "@/ui/auth/redirect";
import { start } from "@/ui/auth/session";
import { useClerk } from "@/ui/clerk";
import { useAction } from "@/ui/useAction";

const schema = z.object({
  firstName: z.string().trim().min(1, "Enter your first name"),
  lastName: z.string().trim().min(1, "Enter your last name"),
  email: z.email("Enter your email address"),
  password: z.string().min(8, "Use at least 8 characters"),
});

const codeSchema = z.object({ code: z.string().trim().min(1, "Enter the code") });

type Step = "details" | "verify";

export interface SignUpProps {
  /** target is where to land after signing up. */
  target: string;
}

/** SignUp creates an account with email and password, Google, or Microsoft. */
export const SignUp = ({ target }: SignUpProps): ReactElement => {
  const clerk = useClerk();
  const [step, setStep] = useState<Step>("details");
  const [oauthError, setOAuthError] = useState<string | null>(null);
  const methods = Form.use({
    values: { firstName: "", lastName: "", email: "", password: "" },
    schema,
  });
  const codeMethods = Form.use({ values: { code: "" }, schema: codeSchema });

  const create = useAction(async () => {
    if (clerk?.client == null || !methods.validate()) return;
    const { firstName, lastName, email, password } = methods.value();
    await clerk.client.signUp.create({
      firstName,
      lastName,
      emailAddress: email,
      password,
    });
    await clerk.client.signUp.prepareEmailAddressVerification({
      strategy: "email_code",
    });
    setStep("verify");
  });

  const verify = useAction(async () => {
    if (clerk?.client == null || !codeMethods.validate()) return;
    const res = await clerk.client.signUp.attemptEmailAddressVerification({
      code: codeMethods.value().code,
    });
    if (res.status !== "complete") throw new Error("That code did not work");
    await start(clerk, res.createdSessionId, target);
  });

  if (step === "verify")
    return (
      <Card
        title="Check your email"
        description={`We sent a code to ${methods.value().email}`}
        error={verify.error}
      >
        <Form.Form<typeof codeSchema> {...codeMethods}>
          <Form.TextField
            path="code"
            label="Code"
            required={false}
            padHelpText={false}
            inputProps={{ autoFocus: true, autoComplete: "one-time-code" }}
          />
          <Button.Button
            variant="filled"
            size="large"
            full="x"
            justify="center"
            onClick={verify.run}
            status={verify.loading ? "loading" : undefined}
            trigger={["Enter"]}
          >
            Verify
          </Button.Button>
        </Form.Form>
      </Card>
    );

  return (
    <Card
      title="Create an account"
      description="Activate machines and manage licenses"
      error={create.error ?? oauthError}
      footer={
        <Text.Text level="small" color={9}>
          Already have an account?{" "}
          <Text.Text
            el="a"
            level="small"
            variant="link"
            href={withTarget("/login", target)}
          >
            Log in
          </Text.Text>
        </Text.Text>
      }
    >
      <OAuth mode="sign-up" target={target} onError={setOAuthError} />
      <Form.Form<typeof schema> {...methods}>
        <Flex.Box x gap="medium">
          <Form.TextField
            path="firstName"
            label="First name"
            required={false}
            padHelpText={false}
            grow
            inputProps={{ autoComplete: "given-name", autoFocus: true }}
          />
          <Form.TextField
            path="lastName"
            label="Last name"
            required={false}
            padHelpText={false}
            grow
            inputProps={{ autoComplete: "family-name" }}
          />
        </Flex.Box>
        <Form.TextField
          path="email"
          label="Email"
          required={false}
          padHelpText={false}
          inputProps={{ type: "email", autoComplete: "email" }}
        />
        <Form.TextField
          path="password"
          label="Password"
          required={false}
          padHelpText={false}
          inputProps={{ type: "password", autoComplete: "new-password" }}
        />
        <div id="clerk-captcha" />
        <Button.Button
          variant="filled"
          size="large"
          full="x"
          justify="center"
          onClick={create.run}
          disabled={clerk == null}
          status={create.loading ? "loading" : undefined}
          trigger={["Enter"]}
        >
          Create account
        </Button.Button>
      </Form.Form>
    </Card>
  );
};
