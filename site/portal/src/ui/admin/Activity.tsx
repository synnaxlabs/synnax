// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useId } from "react";

import { type Activity as ActivityData } from "@/page";
import { type Category } from "@/server/event";
import { ActivityTable } from "@/ui/admin/ActivityTable";
import { navigate } from "@/ui/api";
import { Page } from "@/ui/Page";

const CATEGORY_LABELS: Record<Category, string> = {
  licenses: "Licenses",
  machines: "Machines",
  members: "Members",
  organizations: "Organizations",
};

const CATEGORIES = Object.keys(CATEGORY_LABELS) as Category[];

export interface ActivityFilter {
  category: Category | null;
  /** staff keeps only what staff members did. */
  staff: boolean;
}

export interface ActivityProps extends ActivityData {
  filter: ActivityFilter;
}

const activityHref = ({ category, staff }: ActivityFilter): string => {
  const params = new URLSearchParams();
  if (category != null) params.set("category", category);
  if (staff) params.set("staff", "1");
  const search = params.toString();
  return search === "" ? "/admin/activity" : `/admin/activity?${search}`;
};

/** Activity is the audit log of every organization. Staff only. */
export const Activity = ({
  page,
  actors,
  filter,
  olderHref,
}: ActivityProps): ReactElement => {
  const staffID = useId();
  return (
    <Page title="Activity" subtitle="Everything done in the portal, newest first">
      <Flex.Box x justify="between" align="center" gap="medium" full="x" wrap>
        <Select.Buttons<Category>
          value={filter.category ?? undefined}
          onChange={(category: Category | null) =>
            navigate(activityHref({ ...filter, category }))
          }
          allowNone
        >
          {CATEGORIES.map((c) => (
            <Select.Item key={c} itemKey={c} size="small">
              {CATEGORY_LABELS[c]}
            </Select.Item>
          ))}
        </Select.Buttons>
        <Flex.Box x align="center" gap="small">
          <Input.Checkbox
            id={staffID}
            value={filter.staff}
            onChange={(staff) => navigate(activityHref({ ...filter, staff }))}
            size="small"
          />
          <Text.Text el="label" htmlFor={staffID} level="small" color={10}>
            Staff actions only
          </Text.Text>
        </Flex.Box>
      </Flex.Box>
      <ActivityTable
        page={page}
        actors={actors}
        olderHref={olderHref}
        filtered={filter.category != null || filter.staff}
        organizations
      />
    </Page>
  );
};
