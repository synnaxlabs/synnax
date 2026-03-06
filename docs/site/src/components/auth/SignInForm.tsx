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

type Step = "identifier" | "password" | "complete";

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

export const SignInForm = (): ReactElement => {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [step, setStep] = useState<Step>("identifier");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const getClerk = useClerk();

  const handleOAuth = useCallback(
    async (strategy: string) => {
      try {
        const clerk = getClerk();
        await clerk.client.signIn.authenticateWithRedirect({
          strategy,
          redirectUrl: "/sign-in/sso-callback",
          redirectUrlComplete: "/",
        });
      } catch (e: any) {
        setError(e.errors?.[0]?.longMessage ?? e.message ?? "OAuth failed");
      }
    },
    [getClerk],
  );

  const handleEmailSubmit = useCallback(async () => {
    if (email.trim() === "") return;
    setLoading(true);
    setError("");
    try {
      const clerk = getClerk();
      const signIn = await clerk.client.signIn.create({ identifier: email });
      if (signIn.status === "complete") {
        await clerk.setActive({ session: signIn.createdSessionId });
        window.location.href = "/";
        return;
      }
      setStep("password");
    } catch (e: any) {
      setError(e.errors?.[0]?.longMessage ?? e.message ?? "Something went wrong");
    } finally {
      setLoading(false);
    }
  }, [email, getClerk]);

  const handlePasswordSubmit = useCallback(async () => {
    if (password === "") return;
    setLoading(true);
    setError("");
    try {
      const clerk = getClerk();
      const signIn = await clerk.client.signIn.attemptFirstFactor({
        strategy: "password",
        password,
      });
      if (signIn.status === "complete") {
        await clerk.setActive({ session: signIn.createdSessionId });
        window.location.href = "/";
      }
    } catch (e: any) {
      setError(e.errors?.[0]?.longMessage ?? e.message ?? "Invalid password");
    } finally {
      setLoading(false);
    }
  }, [password, getClerk]);

  return (
    <Flex.Box
      direction="y"
      align="center"
      gap={6}
      style={{ width: "100%", maxWidth: "380px" }}
    >
      <Text.Text level="h2" weight={500}>
        {step === "identifier" ? "Sign in to Synnax" : "Enter your password"}
      </Text.Text>

      {step === "identifier" && (
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

          <Button.Button
            variant="filled"
            onClick={() => void handleEmailSubmit()}
            disabled={loading || email.trim() === ""}
            style={{ width: "100%" }}
            size="large"
          >
            {loading ? "Continuing..." : "Continue"}
          </Button.Button>
        </>
      )}

      {step === "password" && (
        <>
          <Text.Text level="small" color={7}>
            {email}
          </Text.Text>

          <Flex.Box direction="y" gap={2} style={{ width: "100%" }}>
            <Text.Text level="small" weight={450} color={9}>
              Password
            </Text.Text>
            <Input.Text
              value={password}
              onChange={setPassword}
              placeholder="Enter your password"
              type="password"
              size="large"
              autoFocus
            />
          </Flex.Box>

          <Button.Button
            variant="filled"
            onClick={() => void handlePasswordSubmit()}
            disabled={loading || password === ""}
            style={{ width: "100%" }}
            size="large"
          >
            {loading ? "Signing in..." : "Sign In"}
          </Button.Button>

          <Button.Button
            variant="text"
            onClick={() => {
              setStep("identifier");
              setPassword("");
              setError("");
            }}
          >
            Back
          </Button.Button>
        </>
      )}

      {error !== "" && <Status.Summary variant="error" message={error} />}

      <Text.Text level="small" color={7}>
        Don&apos;t have an account?{" "}
        <a href="/sign-up" style={{ color: "var(--pluto-primary-z)" }}>
          Sign up
        </a>
      </Text.Text>
    </Flex.Box>
  );
};
