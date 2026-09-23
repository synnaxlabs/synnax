// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Status, Text } from "@synnaxlabs/pluto";
import { navigate } from "astro:transitions/client";
import { type ReactElement, useEffect, useState } from "react";

import { Card } from "@/portal/ui/auth/Card";
import { target } from "@/portal/ui/auth/redirect";
import { errorMessage, useClerk } from "@/portal/ui/clerk";

/** SSOCallback completes a Google or Microsoft sign-in and sends the user on. */
export const SSOCallback = (): ReactElement => {
  const clerk = useClerk();
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (clerk == null) return;
    clerk
      .handleRedirectCallback(
        {
          signInFallbackRedirectUrl: target(),
          signUpFallbackRedirectUrl: target(),
        },
        (to) => navigate(to),
      )
      .catch((err: unknown) => setError(errorMessage(err)));
  }, [clerk]);
  return (
    <Card
      title="Signing you in"
      error={error}
      footer={
        error != null && (
          <Text.Text el="a" level="small" variant="link" href="/sign-in">
            Back to sign in
          </Text.Text>
        )
      }
    >
      <Flex.Box align="center" justify="center" style={{ padding: "2rem" }}>
        {error == null && <Status.Indicator variant="loading" />}
      </Flex.Box>
    </Card>
  );
};
