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
import { navigate } from "astro:transitions/client";
import { type ReactElement, useState } from "react";
import { z } from "zod";

import { Card } from "@/ui/auth/Card";
import { OAuth } from "@/ui/auth/OAuth";
import { withTarget } from "@/ui/auth/redirect";
import { type Clerk, useClerk } from "@/ui/clerk";
import { useAction } from "@/ui/useAction";

const schema = z.object({
  email: z.email("Enter your email address"),
  password: z.string().min(1, "Enter your password"),
});

const codeSchema = z.object({ code: z.string().trim().min(1, "Enter the code") });

type Step = "credentials" | "second-factor";

const finish = async (
  clerk: Clerk,
  target: string,
  sessionId: string | null,
): Promise<void> => {
  if (sessionId == null) throw new Error("Login did not start a session");
  await clerk.setActive({ session: sessionId });
  await navigate(target);
};

export interface SignInProps {
  /** target is where to land after signing in. */
  target: string;
}

/** SignIn signs a user in with email and password, Google, or Microsoft. */
export const SignIn = ({ target }: SignInProps): ReactElement => {
  const clerk = useClerk();
  const [step, setStep] = useState<Step>("credentials");
  const [oauthError, setOAuthError] = useState<string | null>(null);
  const methods = Form.use({ values: { email: "", password: "" }, schema });
  const codeMethods = Form.use({ values: { code: "" }, schema: codeSchema });

  const signIn = useAction(async () => {
    if (clerk?.client == null || !methods.validate()) return;
    const { email, password } = methods.value();
    const res = await clerk.client.signIn.create({ identifier: email, password });
    if (res.status === "complete")
      return await finish(clerk, target, res.createdSessionId);
    if (res.status === "needs_second_factor") {
      setStep("second-factor");
      return;
    }
    throw new Error(`Sign-in needs ${res.status ?? "another step"}`);
  });

  const verify = useAction(async () => {
    if (clerk?.client == null || !codeMethods.validate()) return;
    const res = await clerk.client.signIn.attemptSecondFactor({
      strategy: "totp",
      code: codeMethods.value().code,
    });
    if (res.status === "complete")
      return await finish(clerk, target, res.createdSessionId);
    throw new Error("That code did not work");
  });

  if (step === "second-factor")
    return (
      <Card
        title="Two-step verification"
        description="Enter the code from your authenticator app."
        error={verify.error}
      >
        <Form.Form<typeof codeSchema> {...codeMethods}>
          <Form.TextField
            path="code"
            label="Code"
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
      title="Sign in"
      description="Manage your Synnax licenses."
      error={signIn.error ?? oauthError}
      footer={
        <Text.Text level="small" color={9}>
          New to Synnax?{" "}
          <Text.Text
            el="a"
            level="small"
            variant="link"
            href={withTarget("/sign-up", target)}
          >
            Create an account
          </Text.Text>
        </Text.Text>
      }
    >
      <OAuth mode="sign-in" target={target} onError={setOAuthError} />
      <Form.Form<typeof schema> {...methods}>
        <Form.TextField
          path="email"
          label="Email"
          inputProps={{ type: "email", autoComplete: "email", autoFocus: true }}
        />
        <Form.TextField
          path="password"
          label="Password"
          inputProps={{ type: "password", autoComplete: "current-password" }}
        />
        <Button.Button
          variant="filled"
          size="large"
          full="x"
          justify="center"
          onClick={signIn.run}
          disabled={clerk == null}
          status={signIn.loading ? "loading" : undefined}
          trigger={["Enter"]}
        >
          Sign in
        </Button.Button>
        <Text.Text
          el="a"
          level="small"
          variant="link"
          href={withTarget("/sign-in/reset", target)}
          className="portal-auth__reset"
        >
          Forgot your password?
        </Text.Text>
      </Form.Form>
    </Card>
  );
};
