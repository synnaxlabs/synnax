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
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { strings } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import type { LicenseList } from "@/page";
import type { Listed } from "@/server/directory";
import type { Query } from "@/server/license/list";
import { type Status, STATUSES } from "@/server/license/status";
import { IssueDialog } from "@/ui/admin/IssueDialog";
import { LicenseTable } from "@/ui/admin/LicenseTable";
import { navigate } from "@/ui/api";
import { STATUS_LABELS } from "@/ui/format";
import { Empty, Page } from "@/ui/Page";

export interface LicensesProps extends LicenseList {
  /** teams are the organizations a license can be issued to. */
  teams: Listed[];
  now: Date;
}

const licensesHref = ({ search, status }: Query): string => {
  const params = new URLSearchParams();
  const trimmed = search?.trim() ?? "";
  if (trimmed !== "") params.set("search", trimmed);
  if (status != null) params.set("status", status);
  const query = params.toString();
  return query === "" ? "/admin/licenses" : `/admin/licenses?${query}`;
};

/** Licenses lists every license a page at a time and issues new ones. Staff only. */
export const Licenses = ({
  teams,
  page,
  query,
  nextHref,
  now,
}: LicensesProps): ReactElement => {
  const [search, setSearch] = useState(query.search ?? "");
  const filtered = query.search != null || query.status != null;
  const firstHref = licensesHref(query);
  return (
    <Page
      title="Licenses"
      subtitle="Every license issued"
      actions={<IssueDialog teams={teams} />}
    >
      {page.total === 0 && !filtered ? (
        <Empty icon={<Icon.Access />} message="No licenses issued yet" />
      ) : (
        <>
          <Flex.Box x justify="between" align="center" gap="medium" full="x">
            <Input.Text
              value={search}
              onChange={setSearch}
              onKeyDown={(e) => {
                if (e.key !== "Enter") return;
                navigate(licensesHref({ search, status: query.status }));
              }}
              placeholder={
                <>
                  <Icon.Search />
                  Label, organization, or key
                </>
              }
              className="portal-search"
            />
            <Flex.Box x align="center" gap="medium">
              <Text.Text level="small" color={9}>
                {strings.count(page.total, "license")}
              </Text.Text>
              <Select.Buttons<Status>
                value={query.status}
                onChange={(status: Status | null) =>
                  navigate(licensesHref({ search, status: status ?? undefined }))
                }
                allowNone
              >
                {STATUSES.map((s) => (
                  <Select.Item key={s} itemKey={s} size="small">
                    {STATUS_LABELS[s]}
                  </Select.Item>
                ))}
              </Select.Buttons>
            </Flex.Box>
          </Flex.Box>
          {page.entries.length === 0 ? (
            <Empty icon={<Icon.Search />} message="No licenses match" />
          ) : (
            <LicenseTable entries={page.entries} now={now} />
          )}
          {(query.after != null || nextHref != null) && (
            <Flex.Box x justify="center" gap="medium">
              {query.after != null && (
                <Button.Button variant="outlined" href={firstHref}>
                  First page
                </Button.Button>
              )}
              {nextHref != null && (
                <Button.Button variant="outlined" href={nextHref}>
                  Next page
                </Button.Button>
              )}
            </Flex.Box>
          )}
        </>
      )}
    </Page>
  );
};
