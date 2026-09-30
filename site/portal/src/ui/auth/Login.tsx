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
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { navigate } from "astro:transitions/client";
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
import { type Clerk, useClerk } from "@/ui/clerk";
import { useAction } from "@/ui/useAction";

const schema = z.object({
  email: z.email("Enter your email address"),
  password: z.string().min(1, "Enter your password"),
});

const codeSchema = z.object({ code: z.string().trim().min(1, "Enter the code") });

const finish = async (
  clerk: Clerk,
  target: string,
  sessionId: string | null,
): Promise<void> => {
  if (sessionId == null) throw new Error("Login did not start a session");
  await clerk.setActive({ session: sessionId });
  await navigate(target);
};

export interface LoginProps {
  /** target is where to land after logging in. */
  target: string;
}

/** Login logs a user in with email and password, Google, or Microsoft. */
export const Login = ({ target }: LoginProps): ReactElement => {
  const clerk = useClerk();
  // factor is the second step Clerk asked for, or null while on the password step.
  const [factor, setFactor] = useState<CodeFactor | null>(null);
  const [backup, setBackup] = useState<CodeFactor | null>(null);
  const [oauthError, setOAuthError] = useState<string | null>(null);
  const methods = Form.use({ values: { email: "", password: "" }, schema });
  const codeMethods = Form.use({ values: { code: "" }, schema: codeSchema });

  const login = useAction(async () => {
    if (clerk?.client == null || !methods.validate()) return;
    const { email, password } = methods.value();
    const res = await clerk.client.signIn.create({ identifier: email, password });
    if (res.status === "complete")
      return await finish(clerk, target, res.createdSessionId);
    if (res.status === "needs_second_factor" || res.status === "needs_client_trust") {
      const next = choose(res.supportedSecondFactors);
      if (next == null)
        throw new Error("This account needs a step the portal cannot do");
      await prepare(res, next);
      setFactor(next);
      setBackup(backupCode(res.supportedSecondFactors));
      return;
    }
    throw new Error(`Login needs ${res.status ?? "another step"}`);
  });

  const verify = useAction(async () => {
    if (clerk?.client == null || factor == null || !codeMethods.validate()) return;
    const res = await clerk.client.signIn.attemptSecondFactor({
      strategy: factor.strategy,
      code: codeMethods.value().code,
    });
    if (res.status === "complete")
      return await finish(clerk, target, res.createdSessionId);
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
      title="Log in"
      description="Manage your Synnax licenses"
      error={login.error ?? oauthError}
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
      <OAuth mode="login" target={target} onError={setOAuthError} />
      <Form.Form<typeof schema> {...methods}>
        <Form.TextField
          path="email"
          label="Email"
          required={false}
          padHelpText={false}
          inputProps={{ type: "email", autoComplete: "email", autoFocus: true }}
        />
        <Form.TextField
          path="password"
          label="Password"
          required={false}
          padHelpText={false}
          inputProps={{ type: "password", autoComplete: "current-password" }}
        />
        <Button.Button
          variant="filled"
          size="large"
          full="x"
          justify="center"
          onClick={login.run}
          disabled={clerk == null}
          status={login.loading ? "loading" : undefined}
          trigger={["Enter"]}
        >
          Log in
          <Icon.Arrow.Right />
        </Button.Button>
        <Text.Text
          el="a"
          level="small"
          variant="link"
          href={withTarget("/login/reset", target)}
          className="portal-auth__reset"
        >
          Forgot your password?
        </Text.Text>
      </Form.Form>
    </Card>
  );
};
