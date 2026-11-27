// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { auth, resource } from "@synnaxlabs/client";
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
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

// A provider need not send a claim that names the account.
const nameOf = ({ label, account }: auth.oidc.Link): string =>
  label === "" ? account : label;

interface RowProps {
  link: auth.oidc.Link;
  /** The name of the provider whose issuer the link holds. */
  label: string;
  onUnlink?: (link: auth.oidc.Link, label: string) => void;
}

const Row = ({ link, label, onUnlink }: RowProps): ReactElement => (
  <Flex.Box x align="center" justify="between">
    <Flex.Box y gap="tiny">
      <Text.Text weight={500}>{label}</Text.Text>
      <Text.Text level="small" color={9}>
        {nameOf(link)}
      </Text.Text>
    </Flex.Box>
    {onUnlink != null && (
      <Button.Button
        variant="text"
        tooltip="Unlink account"
        onClick={() => onUnlink(link, label)}
      >
        <Icon.Delete />
      </Button.Button>
    )}
  </Flex.Box>
);

interface ListProps {
  subject: resource.ID;
  providers: auth.OIDCLogin[];
  linking: boolean;
  onLink?: (login: auth.OIDCLogin) => void;
  onUnlink?: (link: auth.oidc.Link, label: string) => void;
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
    ({ issuer }) => links.data?.every((l) => l.issuer !== issuer) === true,
  );
  const offered = onLink != null && unlinked.length > 0;
  // a link whose provider left the Core's settings shows its issuer
  const labelOf = ({ issuer }: auth.oidc.Link): string =>
    providers.find((p) => p.issuer === issuer)?.provider ?? issuer;
  return (
    <>
      <Flex.Box y className={CSS.BE("credential", "keys")}>
        {links.variant === "error" && (
          <Text.Text status="error">{links.status.message}</Text.Text>
        )}
        {links.data?.map((l) => (
          <Row key={l.key} link={l} label={labelOf(l)} onUnlink={onUnlink} />
        ))}
        {links.data?.length === 0 && (
          <Text.Text level="small" color={9}>
            No linked accounts
          </Text.Text>
        )}
      </Flex.Box>
      {offered && !window.isSecureContext && (
        <Status.Summary
          variant="warning"
          message="Linking an account needs HTTPS"
          description="The browser can only reach a provider on a page served over HTTPS. Open the Console over HTTPS, or use the desktop Console."
        />
      )}
      {offered &&
        window.isSecureContext &&
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
  subject: resource.ID;
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
  const providers = Core.useProviders(client?.params, true) ?? [];
  const selectedCore = Session.Core.useSelectSelectedKey();
  const query = { subject };
  const invalidate = Credential.OIDC.useInvalidate();
  const refresh = Credential.OIDC.useRefresh();
  // This read has no live updates, so a change made here fetches it again.
  const handleChange = (message: string) => {
    refresh(query);
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
  const reportUnfinished = Core.useReportUnfinished();
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
      if (failure != null) reportUnfinished(failure, "Link not finished");
    }, "Failed to link account");
  };
  const confirm = Modals.useConfirm();
  const handleUnlink = (link: auth.oidc.Link, label: string) => {
    onMessage(null);
    // a session opened through the link ends with it
    const current = client?.auth.session?.credential === link.key;
    handleError(async () => {
      const confirmed = await confirm({
        message: `Are you sure you want to unlink ${nameOf(link)}?`,
        description: current
          ? `You logged in with this ${label} account, so unlinking it ends this session. You can no longer log in with it.`
          : `This user can no longer log in with this ${label} account.`,
        confirm: { variant: "error", label: "Unlink" },
        cancel: { label: "Cancel" },
      });
      if (confirmed === true) unlink.update(link.key);
    }, "Failed to unlink account");
  };
  // Drop the answer on unmount so the next mount shows changes made elsewhere.
  useEffect(() => () => invalidate(query), []);
  return (
    <Flex.Box y>
      <Text.Text level="h5">Provider accounts</Text.Text>
      <List
        subject={subject}
        providers={providers}
        linking={linking}
        onLink={own ? handleLink : undefined}
        onUnlink={canUnlink ? handleUnlink : undefined}
      />
    </Flex.Box>
  );
};
