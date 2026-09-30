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
import { OAuth } from "@/ui/auth/OAuth";
import { withTarget } from "@/ui/auth/redirect";
import {
  backupCode,
  choose,
  type CodeFactor,
  prepare,
  prompt,
} from "@/ui/auth/secondFactor";
import { start } from "@/ui/auth/session";
import { useClerk } from "@/ui/clerk";
import { useAction } from "@/ui/useAction";

const schema = z.object({
  email: z.email("Enter your email address"),
  password: z.string().min(1, "Enter your password"),
});

const codeSchema = z.object({ code: z.string().trim().min(1, "Enter the code") });

export interface SignInProps {
  /** target is where to land after signing in. */
  target: string;
}

/** SignIn signs a user in with email and password, Google, or Microsoft. */
export const SignIn = ({ target }: SignInProps): ReactElement => {
  const clerk = useClerk();
  // factor is the second step Clerk asked for, or null while on the password step.
  const [factor, setFactor] = useState<CodeFactor | null>(null);
  const [backup, setBackup] = useState<CodeFactor | null>(null);
  const [oauthError, setOAuthError] = useState<string | null>(null);
  const methods = Form.use({ values: { email: "", password: "" }, schema });
  const codeMethods = Form.use({ values: { code: "" }, schema: codeSchema });

  const signIn = useAction(async () => {
    if (clerk?.client == null || !methods.validate()) return;
    const { email, password } = methods.value();
    const res = await clerk.client.signIn.create({ identifier: email, password });
    if (res.status === "complete")
      return await start(clerk, res.createdSessionId, target);
    if (res.status === "needs_second_factor" || res.status === "needs_client_trust") {
      const next = choose(res.supportedSecondFactors);
      if (next == null)
        throw new Error("This account needs a step the portal cannot do");
      await prepare(res, next);
      setFactor(next);
      setBackup(backupCode(res.supportedSecondFactors));
      return;
    }
    throw new Error(`Sign-in needs ${res.status ?? "another step"}`);
  });

  const verify = useAction(async () => {
    if (clerk?.client == null || factor == null || !codeMethods.validate()) return;
    const res = await clerk.client.signIn.attemptSecondFactor({
      strategy: factor.strategy,
      code: codeMethods.value().code,
    });
    if (res.status === "complete")
      return await start(clerk, res.createdSessionId, target);
    throw new Error("That code did not work");
  });

  if (factor != null)
    return (
      <Card
        title="Two-step verification"
        description={prompt(factor)}
        error={verify.error}
        footer={
          backup != null &&
          factor !== backup && (
            <Text.Text
              el="button"
              level="small"
              variant="link"
              onClick={() => setFactor(backup)}
            >
              Use a backup code
            </Text.Text>
          )
        }
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
