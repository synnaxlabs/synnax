// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { type Entry, type Stage } from "@/server/referral";
import { referralPath, userPath } from "@/shell";
import { ReferralDialog } from "@/ui/admin/ReferralDialog";
import { Fact, Facts } from "@/ui/Facts";
import { date } from "@/ui/format";
import { Empty, Page } from "@/ui/Page";
import { Row, Table } from "@/ui/Table";

export interface ReferralsProps {
  entries: Entry[];
}

const COLUMNS = "minmax(0, 2fr) 14rem 16rem 18rem 16rem 7rem";

/** STAGES are the stages in the order a recipient goes through them. */
const STAGES: Stage[] = ["sent", "opened", "signed_up", "installed"];

const STAGE_LABELS: Record<Stage, string> = {
  sent: "Sent",
  opened: "Opened",
  signed_up: "Signed up",
  installed: "Installed",
};

const STAGE_VARIANTS: Record<Stage, "disabled" | "info" | "success"> = {
  sent: "disabled",
  opened: "info",
  signed_up: "info",
  installed: "success",
};

/** reached says whether the recipient of a referral got at least as far as `stage`. */
const reached = (e: Entry, stage: Stage): boolean =>
  STAGES.indexOf(e.stage) >= STAGES.indexOf(stage);

const StageTag = ({ stage }: { stage: Stage }): ReactElement => (
  <Tag.Tag icon={<Status.Indicator variant={STAGE_VARIANTS[stage]} />} size="small">
    {STAGE_LABELS[stage]}
  </Tag.Tag>
);

/**
 * Referrals lists every referral link with how far its recipient got, and makes new
 * ones. Staff only.
 */
export const Referrals = ({ entries }: ReferralsProps): ReactElement => (
  <Page
    title="Referrals"
    subtitle="Links that guide someone through Synnax Desktop setup"
    actions={<ReferralDialog />}
  >
    {entries.length === 0 ? (
      <Empty icon={<Icon.Link />} message="No referral links yet" />
    ) : (
      <>
        <Facts>
          <Fact label="Links" value={String(entries.length)} />
          {STAGES.slice(1).map((s) => (
            <Fact
              key={s}
              label={STAGE_LABELS[s]}
              value={String(entries.filter((e) => reached(e, s)).length)}
            />
          ))}
        </Facts>
        <Table
          columns={COLUMNS}
          head={["For", "Stage", "Last active", "Referred by", "Made", ""]}
        >
          {entries.map((e) => (
            <ReferralRow key={e.referral.key} {...e} />
          ))}
        </Table>
      </>
    )}
  </Page>
);

const ReferralRow = ({
  referral: ref,
  referrer,
  account,
  usage,
  stage,
}: Entry): ReactElement => (
  <Row columns={COLUMNS}>
    <Flex.Box y gap="tiny" className="portal-list__stack">
      <Text.Text level="p" weight={500} overflow="fade">
        {ref.recipient}
      </Text.Text>
      {account != null && (
        <Text.Text level="small" color={9} overflow="fade">
          {account.email}
        </Text.Text>
      )}
    </Flex.Box>
    <Flex.Box>
      <StageTag stage={stage} />
    </Flex.Box>
    <Text.Text level="p" color={9}>
      {date(usage.lastActive)}
    </Text.Text>
    <Text.Text level="p" color={9} overflow="fade">
      {referrer}
    </Text.Text>
    <Text.Text level="p" color={9}>
      {date(ref.createdAt)}
    </Text.Text>
    <Flex.Box x justify="end" gap="tiny">
      <Button.Copy
        variant="text"
        size="small"
        text={() => new URL(referralPath(ref.key), window.location.origin).href}
        tooltip="Copy the link"
        aria-label="Copy referral link"
      />
      {account != null && (
        <Button.Button
          variant="text"
          size="small"
          href={userPath(account.userID)}
          tooltip="Open the user"
          aria-label="Open the user"
        >
          <Icon.User />
        </Button.Button>
      )}
    </Flex.Box>
  </Row>
);
