// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Status, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";

import { post, save } from "@/portal/ui/api";
import { Card } from "@/portal/ui/auth/Card";
import { activateURL, type Linked } from "@/portal/ui/desktop/url";
import { useAction } from "@/portal/ui/useAction";

/** TOKEN_FILE is what the token downloads as when the browser cannot open the app. */
const TOKEN_FILE = "synnax-desktop.license";

export interface LinkProps {
  state: string;
  fingerprint: string[];
  /** name is the hostname the app reported; shown so the person knows what links. */
  name: string;
  email: string;
  /** problem is why the page cannot link, when the query did not come from the app. */
  problem: string | null;
}

/**
 * Link is the page the Desktop app opens in the browser: confirm the machine, issue
 * its license, and hand the token back through the app's URL scheme.
 */
export const Link = ({
  state,
  fingerprint,
  name,
  email,
  problem,
}: LinkProps): ReactElement => {
  const [linked, setLinked] = useState<Linked | null>(null);
  const [url, setURL] = useState<string | null>(null);
  const action = useAction(
    useCallback(async () => {
      const res = await post<Linked>("/api/portal/desktop/link", {
        fingerprint,
        name,
      });
      const target = activateURL(state, res);
      setLinked(res);
      setURL(target);
      window.location.assign(target);
    }, [state, fingerprint, name]),
  );
  if (problem != null)
    return (
      <Card title="Cannot sign in this machine" error={problem}>
        <Text.Text level="p" color={10}>
          Open Synnax Desktop and choose Sign in again.
        </Text.Text>
      </Card>
    );
  if (linked != null && url != null)
    return (
      <Card
        title="Return to Synnax Desktop"
        description={`${name} is linked to ${email}. The app should open on its own.`}
        footer={
          <Button.Button
            variant="text"
            size="small"
            onClick={() => save(new Blob([linked.token]), TOKEN_FILE)}
          >
            Download a license file instead
          </Button.Button>
        }
      >
        <Button.Button variant="filled" onClick={() => window.location.assign(url)}>
          Open Synnax Desktop
        </Button.Button>
      </Card>
    );
  return (
    <Card
      title="Sign in to Synnax Desktop"
      description={`Link ${name} to ${email}.`}
      error={action.error}
      footer={
        <Text.Text level="small" color={9}>
          You can unlink this machine from the portal at any time.
        </Text.Text>
      }
    >
      <Flex.Box y gap="small">
        <Button.Button
          variant="filled"
          status={action.loading ? "loading" : undefined}
          onClick={action.run}
        >
          Continue
        </Button.Button>
        <Status.Summary
          variant="info"
          level="small"
          message="Synnax Desktop opens once the machine is linked."
        />
      </Flex.Box>
    </Card>
  );
};
