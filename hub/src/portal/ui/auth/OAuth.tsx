// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Divider, Flex, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";

import { target } from "@/portal/ui/auth/redirect";
import { errorMessage, useClerk } from "@/portal/ui/clerk";

type Strategy = "oauth_google" | "oauth_microsoft";

const GoogleMark = (): ReactElement => (
  <svg width="18" height="18" viewBox="0 0 48 48" aria-hidden="true">
    <path
      fill="#EA4335"
      d="M24 9.5c3.5 0 6.6 1.2 9 3.5l6.7-6.7C35.6 2.5 30.2 0 24 0 14.6 0 6.5 5.4 2.6 13.3l7.8 6.1C12.3 13.6 17.7 9.5 24 9.5z"
    />
    <path
      fill="#4285F4"
      d="M46.5 24.5c0-1.6-.1-3.1-.4-4.5H24v9h12.7c-.6 3-2.3 5.5-4.8 7.2l7.5 5.8c4.4-4.1 7.1-10.1 7.1-17.5z"
    />
    <path
      fill="#FBBC05"
      d="M10.4 28.6A14.5 14.5 0 0 1 9.5 24c0-1.6.3-3.2.8-4.6l-7.8-6.1A24 24 0 0 0 0 24c0 3.9.9 7.5 2.6 10.7l7.8-6.1z"
    />
    <path
      fill="#34A853"
      d="M24 48c6.5 0 11.9-2.1 15.9-5.8l-7.5-5.8c-2.1 1.4-4.9 2.3-8.4 2.3-6.3 0-11.7-4.1-13.6-9.9l-7.8 6.1C6.5 42.6 14.6 48 24 48z"
    />
  </svg>
);

const MicrosoftMark = (): ReactElement => (
  <svg width="18" height="18" viewBox="0 0 23 23" aria-hidden="true">
    <rect x="1" y="1" width="10" height="10" fill="#F25022" />
    <rect x="12" y="1" width="10" height="10" fill="#7FBA00" />
    <rect x="1" y="12" width="10" height="10" fill="#00A4EF" />
    <rect x="12" y="12" width="10" height="10" fill="#FFB900" />
  </svg>
);

const PROVIDERS: { strategy: Strategy; name: string; mark: ReactElement }[] = [
  { strategy: "oauth_google", name: "Google", mark: <GoogleMark /> },
  { strategy: "oauth_microsoft", name: "Microsoft", mark: <MicrosoftMark /> },
];

export interface OAuthProps {
  /** mode picks whether the redirect starts a sign-in or a sign-up. */
  mode: "sign-in" | "sign-up";
  onError: (message: string) => void;
}

/** OAuth renders the Google and Microsoft buttons and a divider under them. */
export const OAuth = ({ mode, onError }: OAuthProps): ReactElement => {
  const clerk = useClerk();
  const [pending, setPending] = useState<Strategy | null>(null);
  const start = useCallback(
    (strategy: Strategy) => {
      if (clerk?.client == null) return;
      setPending(strategy);
      const params = {
        strategy,
        redirectUrl: "/sso-callback",
        redirectUrlComplete: target(),
      };
      const flow =
        mode === "sign-in"
          ? clerk.client.signIn.authenticateWithRedirect(params)
          : clerk.client.signUp.authenticateWithRedirect(params);
      flow.catch((err: unknown) => {
        setPending(null);
        onError(errorMessage(err));
      });
    },
    [clerk, mode, onError],
  );
  return (
    <>
      <Flex.Box y gap="small">
        {PROVIDERS.map((p) => (
          <Button.Button
            key={p.strategy}
            variant="outlined"
            size="large"
            full="x"
            justify="center"
            disabled={clerk == null}
            status={pending === p.strategy ? "loading" : undefined}
            onClick={() => start(p.strategy)}
          >
            {p.mark}
            Continue with {p.name}
          </Button.Button>
        ))}
      </Flex.Box>
      <Flex.Box x align="center" gap="medium">
        <Divider.Divider x grow />
        <Text.Text level="small" color={9}>
          or
        </Text.Text>
        <Divider.Divider x grow />
      </Flex.Box>
    </>
  );
};
