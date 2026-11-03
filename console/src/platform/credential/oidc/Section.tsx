// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type auth, type ontology } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Credential, Synnax } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Session } from "@/session";

interface RowProps {
  link: auth.oidc.Link;
  onUnlink?: (key: auth.oidc.LinkKey) => void;
}

const Row = ({ link, onUnlink }: RowProps): ReactElement => (
  <Flex.Box x align="center" justify="between">
    <Flex.Box y gap="tiny">
      <Text.Text weight={500}>{link.provider}</Text.Text>
      <Text.Text level="small" color={9}>
        {link.account}
      </Text.Text>
    </Flex.Box>
    {onUnlink != null && (
      <Button.Button
        variant="text"
        tooltip="Unlink account"
        onClick={() => onUnlink(link.key)}
      >
        <Icon.Delete />
      </Button.Button>
    )}
  </Flex.Box>
);

interface ListProps {
  subject: ontology.ID;
  providers: auth.OIDCLogin[];
  linking: boolean;
  onLink: (login: auth.OIDCLogin) => void;
  onUnlink?: (key: auth.oidc.LinkKey) => void;
}

const List = ({
  subject,
  providers,
  linking,
  onLink,
  onUnlink,
}: ListProps): ReactElement => {
  const links = Credential.OIDC.useResult({ subject });
  const unlinked = providers.filter(
    ({ provider }) => links.data?.every((l) => l.provider !== provider) === true,
  );
  return (
    <>
      <Flex.Box y className={CSS.BE("credential", "keys")}>
        {links.variant === "error" && (
          <Text.Text status="error">{links.status.message}</Text.Text>
        )}
        {links.data?.map((l) => (
          <Row key={l.key} link={l} onUnlink={onUnlink} />
        ))}
        {links.data?.length === 0 && (
          <Text.Text level="small" color={9}>
            No linked accounts
          </Text.Text>
        )}
      </Flex.Box>
      {unlinked.length > 0 && !window.isSecureContext && (
        <Status.Summary
          variant="warning"
          message="Linking an account needs HTTPS"
          description="The browser can only reach a provider on a page served over HTTPS. Open the Console over HTTPS, or use the desktop Console."
        />
      )}
      {window.isSecureContext &&
        unlinked.map((login) => (
          <Button.Button
            key={login.provider}
            variant="outlined"
            full="x"
            justify="center"
            disabled={linking}
            onClick={() => onLink(login)}
          >
            Link {login.provider}
          </Button.Button>
        ))}
    </>
  );
};

export interface SectionProps {
  /** The user whose provider accounts to manage. */
  subject: ontology.ID;
  /** Receives the result of a change, or null when a new one starts. */
  onMessage: (message: string | null) => void;
}

/**
 * Lists a user's provider accounts. The user can link more of their own, and the
 * caller can unlink them when it is that user or may delete that user.
 */
export const Section = ({ subject, onMessage }: SectionProps): ReactElement => {
  const client = Synnax.use();
  // The Core links a proven account to the caller alone.
  const own = client?.auth.subject?.key === subject.key;
  const canUnlink = Access.useDeleteGranted(subject) || own;
  const providers = Core.useProviders(client?.params, own);
  const selectedCore = Session.Core.useSelectSelectedKey();
  const query = { subject };
  const invalidate = Credential.OIDC.useInvalidate();
  // This read has no live updates, so a remount is what fetches it again.
  const [generation, setGeneration] = useState(0);
  const handleChange = (message: string) => {
    invalidate(query);
    setGeneration((g) => g + 1);
    onMessage(message);
  };
  const link = Credential.OIDC.useLink({
    afterSuccess: () => handleChange("Successfully linked account"),
  });
  const unlink = Credential.OIDC.useUnlink({
    afterSuccess: () => handleChange("Successfully unlinked account"),
  });
  // a second link would replace the first one's loopback listener
  const [linking, setLinking] = useState(false);
  const handleError = Status.useErrorHandler();
  const startLink = async (login: auth.OIDCLogin): Promise<void> => {
    if (Session.Runtime.ENGINE === "tauri") {
      await link.updateAsync(await Core.authorizeDesktop(login));
      return;
    }
    if (selectedCore == null) throw new Error("No Core is selected");
    await Core.beginBrowserLogin(selectedCore, login, subject.key);
  };
  const handleLink = (login: auth.OIDCLogin) => {
    onMessage(null);
    setLinking(true);
    handleError(async () => {
      let failure: Error | undefined;
      try {
        await startLink(login);
      } catch (e) {
        failure = errors.fromUnknown(e);
      }
      setLinking(false);
      if (failure != null) throw failure;
    }, "Failed to link provider account");
  };
  const handleUnlink = (key: auth.oidc.LinkKey) => {
    onMessage(null);
    unlink.update(key);
  };
  useEffect(() => () => invalidate(query), []);
  return (
    <Flex.Box y>
      <Text.Text level="h5">Provider accounts</Text.Text>
      <List
        key={generation}
        subject={subject}
        providers={providers}
        linking={linking}
        onLink={handleLink}
        onUnlink={canUnlink ? handleUnlink : undefined}
      />
    </Flex.Box>
  );
};
