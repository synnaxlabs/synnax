// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement, useState } from "react";

import { type Listed } from "@/server/directory";
import { type Held, type Owned } from "@/server/license/list";
import { IssueDialog } from "@/ui/admin/IssueDialog";
import { LicenseTable } from "@/ui/admin/LicenseTable";
import { type LicenseStatus, STATUS_LABELS, STATUSES, statusOf } from "@/ui/format";
import { Empty, Page } from "@/ui/Page";

export interface LicensesProps {
  /** teams are the organizations a license can be issued to. */
  teams: Listed[];
  licenses: (Owned & Held)[];
  now: Date;
}

const expiry = ({ license: lic }: Owned): number =>
  lic.expiresAt == null ? Infinity : lic.expiresAt.getTime();

/** Licenses lists every license and issues new ones. Staff only. */
export const Licenses = ({ teams, licenses, now }: LicensesProps): ReactElement => {
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<LicenseStatus | null>(null);
  const query = search.trim().toLowerCase();
  const matches = (row: Owned): boolean => {
    const { license: lic, organization: org } = row;
    if (status != null && statusOf(lic, now) !== status) return false;
    if (query === "") return true;
    return `${lic.label} ${org.name} ${lic.key}`.toLowerCase().includes(query);
  };
  const shown = licenses.filter(matches);
  // Soonest first is the only order that answers what the Expiring filter asks.
  if (status === "expiring") shown.sort((a, b) => expiry(a) - expiry(b));
  return (
    <Page
      title="Licenses"
      subtitle="Every license issued"
      actions={<IssueDialog teams={teams} />}
    >
      {licenses.length === 0 ? (
        <Empty icon={<Icon.Access />} message="No licenses issued yet" />
      ) : (
        <>
          <Flex.Box x justify="between" align="center" gap="medium" full="x">
            <Input.Text
              value={search}
              onChange={setSearch}
              placeholder={
                <>
                  <Icon.Search />
                  Label, organization, or key
                </>
              }
              className="portal-search"
            />
            <Select.Buttons<LicenseStatus>
              value={status ?? undefined}
              onChange={setStatus}
              allowNone
            >
              {STATUSES.map((s) => (
                <Select.Item key={s} itemKey={s} size="small">
                  {STATUS_LABELS[s]}
                </Select.Item>
              ))}
            </Select.Buttons>
          </Flex.Box>
          {shown.length === 0 ? (
            <Empty icon={<Icon.Search />} message="No licenses match" />
          ) : (
            <LicenseTable entries={shown} now={now} />
          )}
        </>
      )}
    </Page>
  );
};
