// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { navigate } from "astro:transitions/client";
import { type ReactElement, useEffect, useState } from "react";

import { Card } from "@/ui/auth/Card";
import { target } from "@/ui/auth/redirect";
import { errorMessage, useClerk } from "@/ui/clerk";

/** SSOCallback completes a Google or Microsoft login and sends the user on. */
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
      title="Logging you in"
      error={error}
      footer={
        error != null && (
          <Text.Text el="a" level="small" variant="link" href="/login">
            Back to log in
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
