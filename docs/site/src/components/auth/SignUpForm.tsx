// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Input, Status, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";

type Step = "form" | "verify";

const OAUTH_PROVIDERS = [
  { strategy: "oauth_github" as const, name: "GitHub" },
  { strategy: "oauth_google" as const, name: "Google" },
];

const useClerk = () => {
  const get = useCallback(() => {
    const clerk = (window as any).Clerk;
    if (clerk == null) throw new Error("Clerk not loaded");
    return clerk;
  }, []);
  return get;
};

export const SignUpForm = (): ReactElement => {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [step, setStep] = useState<Step>("form");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const getClerk = useClerk();

  const handleOAuth = useCallback(
    async (strategy: string) => {
      try {
        const clerk = getClerk();
        await clerk.client.signUp.authenticateWithRedirect({
          strategy,
          redirectUrl: "/sign-up/sso-callback",
          redirectUrlComplete: "/",
        });
      } catch (e: any) {
        setError(e.errors?.[0]?.longMessage ?? e.message ?? "OAuth failed");
      }
    },
    [getClerk],
  );

  const handleSubmit = useCallback(async () => {
    if (email.trim() === "" || password === "") return;
    setLoading(true);
    setError("");
    try {
      const clerk = getClerk();
      const signUp = await clerk.client.signUp.create({
        emailAddress: email,
        password,
      });
      if (signUp.status === "complete") {
        await clerk.setActive({ session: signUp.createdSessionId });
        window.location.href = "/";
        return;
      }
      await signUp.prepareEmailAddressVerification({ strategy: "email_code" });
      setStep("verify");
    } catch (e: any) {
      setError(e.errors?.[0]?.longMessage ?? e.message ?? "Something went wrong");
    } finally {
      setLoading(false);
    }
  }, [email, password, getClerk]);

  const handleVerify = useCallback(async () => {
    if (code === "") return;
    setLoading(true);
    setError("");
    try {
      const clerk = getClerk();
      const signUp = await clerk.client.signUp.attemptEmailAddressVerification({
        code,
      });
      if (signUp.status === "complete") {
        await clerk.setActive({ session: signUp.createdSessionId });
        window.location.href = "/";
      }
    } catch (e: any) {
      setError(e.errors?.[0]?.longMessage ?? e.message ?? "Invalid code");
    } finally {
      setLoading(false);
    }
  }, [code, getClerk]);

  return (
    <Flex.Box
      direction="y"
      align="center"
      gap={6}
      style={{ width: "100%", maxWidth: "380px" }}
    >
      <Text.Text level="h2" weight={500}>
        {step === "form" ? "Create your account" : "Verify your email"}
      </Text.Text>

      {step === "form" && (
        <>
          <Flex.Box direction="x" gap={3} style={{ width: "100%" }}>
            {OAUTH_PROVIDERS.map(({ strategy, name }) => (
              <Button.Button
                key={strategy}
                variant="outlined"
                onClick={() => void handleOAuth(strategy)}
                style={{ flex: 1 }}
              >
                {name}
              </Button.Button>
            ))}
          </Flex.Box>

          <Flex.Box
            direction="x"
            align="center"
            gap={3}
            style={{ width: "100%" }}
          >
            <div style={{ flex: 1, height: "1px", background: "var(--pluto-gray-l4)" }} />
            <Text.Text level="small" color={7}>or</Text.Text>
            <div style={{ flex: 1, height: "1px", background: "var(--pluto-gray-l4)" }} />
          </Flex.Box>

          <Flex.Box direction="y" gap={2} style={{ width: "100%" }}>
            <Text.Text level="small" weight={450} color={9}>
              Email address
            </Text.Text>
            <Input.Text
              value={email}
              onChange={setEmail}
              placeholder="you@example.com"
              size="large"
              autoFocus
            />
          </Flex.Box>

          <Flex.Box direction="y" gap={2} style={{ width: "100%" }}>
            <Text.Text level="small" weight={450} color={9}>
              Password
            </Text.Text>
            <Input.Text
              value={password}
              onChange={setPassword}
              placeholder="Create a password"
              type="password"
              size="large"
            />
          </Flex.Box>

          <Button.Button
            variant="filled"
            onClick={() => void handleSubmit()}
            disabled={loading || email.trim() === "" || password === ""}
            style={{ width: "100%" }}
            size="large"
          >
            {loading ? "Creating account..." : "Continue"}
          </Button.Button>
        </>
      )}

      {step === "verify" && (
        <>
          <Text.Text level="small" color={7}>
            We sent a code to {email}
          </Text.Text>

          <Flex.Box direction="y" gap={2} style={{ width: "100%" }}>
            <Text.Text level="small" weight={450} color={9}>
              Verification code
            </Text.Text>
            <Input.Text
              value={code}
              onChange={setCode}
              placeholder="Enter code"
              size="large"
              autoFocus
            />
          </Flex.Box>

          <Button.Button
            variant="filled"
            onClick={() => void handleVerify()}
            disabled={loading || code === ""}
            style={{ width: "100%" }}
            size="large"
          >
            {loading ? "Verifying..." : "Verify"}
          </Button.Button>
        </>
      )}

      {error !== "" && <Status.Summary variant="error" message={error} />}

      {step === "form" && (
        <Text.Text level="small" color={7}>
          Already have an account?{" "}
          <a href="/sign-in" style={{ color: "var(--pluto-primary-z)" }}>
            Sign in
          </a>
        </Text.Text>
      )}
    </Flex.Box>
  );
};
