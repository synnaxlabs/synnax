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
import { withTarget } from "@/ui/auth/redirect";
import { errorMessage, useClerk } from "@/ui/clerk";

export interface SSOCallbackProps {
  /** target is where to land once the sign-in completes. */
  target: string;
}

/** SSOCallback completes a Google or Microsoft sign-in and sends the user on. */
export const SSOCallback = ({ target }: SSOCallbackProps): ReactElement => {
  const clerk = useClerk();
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (clerk == null) return;
    clerk
      .handleRedirectCallback(
        {
          signInFallbackRedirectUrl: target,
          signUpFallbackRedirectUrl: target,
        },
        (to) => navigate(to),
      )
      .catch((err: unknown) => setError(errorMessage(err)));
  }, [clerk, target]);
  return (
    <Card
      title="Signing you in"
      error={error}
      footer={
        error == null ? undefined : (
          <Text.Text
            el="a"
            level="small"
            variant="link"
            href={withTarget("/sign-in", target)}
          >
            Back to sign in
          </Text.Text>
        )
      }
    >
      <Flex.Box align="center" justify="center" className="portal-auth__pending">
        {error == null && <Status.Indicator variant="loading" />}
      </Flex.Box>
    </Card>
  );
};
