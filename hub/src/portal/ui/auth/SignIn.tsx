// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Form, Input, Text } from "@synnaxlabs/pluto";
import { navigate } from "astro:transitions/client";
import { type ReactElement, useCallback, useState } from "react";
import { z } from "zod";

import { Card } from "@/portal/ui/auth/Card";
import { OAuth } from "@/portal/ui/auth/OAuth";
import { target, withTarget } from "@/portal/ui/auth/redirect";
import { errorMessage, useClerk } from "@/portal/ui/clerk";
import { useAction } from "@/portal/ui/useAction";

const schema = z.object({
  email: z.email("Enter your email address"),
  password: z.string().min(1, "Enter your password"),
});

const codeSchema = z.object({ code: z.string().trim().min(1, "Enter the code") });

type Step = "credentials" | "second-factor";

/** SignIn signs a user in with email and password, Google, or Microsoft. */
export const SignIn = (): ReactElement => {
  const clerk = useClerk();
  const [step, setStep] = useState<Step>("credentials");
  const [oauthError, setOAuthError] = useState<string | null>(null);
  const methods = Form.use({ values: { email: "", password: "" }, schema });
  const codeMethods = Form.use({ values: { code: "" }, schema: codeSchema });

  const finish = useCallback(
    async (sessionId: string | null) => {
      if (clerk == null || sessionId == null) return;
      await clerk.setActive({ session: sessionId });
      await navigate(target());
    },
    [clerk],
  );

  const signIn = useAction(
    useCallback(async () => {
      if (clerk?.client == null || !methods.validate()) return;
      const { email, password } = methods.value();
      try {
        const res = await clerk.client.signIn.create({
          identifier: email,
          password,
        });
        if (res.status === "complete") return await finish(res.createdSessionId);
        if (res.status === "needs_second_factor") {
          setStep("second-factor");
          return;
        }
        throw new Error(`Sign-in needs ${res.status ?? "another step"}`);
      } catch (err) {
        throw new Error(errorMessage(err), { cause: err });
      }
    }, [clerk, methods, finish]),
  );

  const verify = useAction(
    useCallback(async () => {
      if (clerk?.client == null || !codeMethods.validate()) return;
      try {
        const res = await clerk.client.signIn.attemptSecondFactor({
          strategy: "totp",
          code: codeMethods.value().code,
        });
        if (res.status === "complete") return await finish(res.createdSessionId);
        throw new Error("That code did not work");
      } catch (err) {
        throw new Error(errorMessage(err), { cause: err });
      }
    }, [clerk, codeMethods, finish]),
  );

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
      description="Manage your Synnax licenses and support."
      error={signIn.error ?? oauthError}
      footer={
        <Text.Text level="small" color={9}>
          New to Synnax?{" "}
          <Text.Text el="a" level="small" variant="link" href={withTarget("/sign-up")}>
            Create an account
          </Text.Text>
        </Text.Text>
      }
    >
      <OAuth mode="sign-in" onError={setOAuthError} />
      <Form.Form<typeof schema> {...methods}>
        <Form.Field<string> path="email" label="Email">
          {(p) => <Input.Text {...p} type="email" autoComplete="email" autoFocus />}
        </Form.Field>
        <Form.Field<string> path="password" label="Password">
          {(p) => <Input.Text {...p} type="password" autoComplete="current-password" />}
        </Form.Field>
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
          href={withTarget("/sign-in/reset")}
          style={{ alignSelf: "center" }}
        >
          Forgot your password?
        </Text.Text>
      </Form.Form>
    </Card>
  );
};
