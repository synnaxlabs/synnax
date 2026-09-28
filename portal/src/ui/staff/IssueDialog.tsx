// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { navigate } from "astro:transitions/client";
import { type ReactElement, useCallback } from "react";
import { z } from "zod";

import { type Term } from "@/server/db/schema";
import { type Listed } from "@/server/directory";
import { post } from "@/ui/api";
import * as Modal from "@/ui/Modal";
import { useAction } from "@/ui/useAction";

const VERSION_PATTERN = /^\d+\.\d+$/;

const schema = z
  .object({
    organization: z.string().min(1, "Choose an organization"),
    label: z.string().trim().min(1, "Give the license a label"),
    term: z.enum(["subscription", "perpetual"]),
    nodes: z.number().int().min(1, "At least one machine"),
    channels: z.number().int().min(0, "0 for unlimited"),
    expiresAt: z.string(),
    maxVersion: z.string().trim(),
  })
  .check((ctx) => {
    const v = ctx.value;
    if (v.term === "subscription" && v.expiresAt === "")
      ctx.issues.push({
        code: "custom",
        message: "A subscription needs an expiry date",
        path: ["expiresAt"],
        input: v.expiresAt,
      });
    if (v.term === "perpetual" && v.maxVersion === "")
      ctx.issues.push({
        code: "custom",
        message: "A perpetual license needs a maximum version",
        path: ["maxVersion"],
        input: v.maxVersion,
      });
    if (v.maxVersion !== "" && !VERSION_PATTERN.test(v.maxVersion))
      ctx.issues.push({
        code: "custom",
        message: "Use major.minor, like 0.62",
        path: ["maxVersion"],
        input: v.maxVersion,
      });
  });

export interface IssueDialogProps {
  /** teams are the organizations in Clerk. Create one there first. */
  teams: Listed[];
}

/** IssueDialog issues a new enterprise license to an organization. Staff only. */
export const IssueDialog = ({ teams }: IssueDialogProps): ReactElement => (
  <Modal.Frame
    name="Issue a license"
    icon={<Icon.Policy />}
    trigger={
      <Dialog.Trigger variant="filled" hideCaret>
        <Icon.Add />
        Issue a license
      </Dialog.Trigger>
    }
  >
    <Content teams={teams} />
  </Modal.Frame>
);

interface OrganizationEntry {
  key: string;
  name: string;
}

const Content = ({ teams }: IssueDialogProps): ReactElement => {
  const entries: OrganizationEntry[] = teams.map((t) => ({
    key: t.clerkOrgID,
    name: t.name,
  }));
  const methods = Form.use({
    values: {
      organization: entries[0]?.key ?? "",
      label: "",
      term: "subscription",
      nodes: 1,
      channels: 0,
      expiresAt: "",
      maxVersion: "",
    },
    schema,
  });
  const action = useAction(
    useCallback(async () => {
      if (!methods.validate()) return;
      const { key } = await post<{ key: string }>("/api/licenses", methods.value());
      await navigate(`/licenses/${key}`);
    }, [methods]),
  );
  return (
    <Form.Form<typeof schema> {...methods}>
      <Modal.Body gap="medium">
        <Form.Field<string>
          path="organization"
          label="Organization"
          helpText="Organizations and their admins are created in the Clerk dashboard."
        >
          {(p) => (
            <Select.Simple<string> {...p} resourceName="Organization">
              {entries.map(({ key, name }) => (
                <Select.Item key={key} itemKey={key}>
                  {name}
                </Select.Item>
              ))}
            </Select.Simple>
          )}
        </Form.Field>
        <Form.TextField
          path="label"
          label="Label"
          inputProps={{ autoFocus: true, placeholder: "Test stand, site B" }}
        />
        <Form.Field<Term> path="term" label="Term">
          {(p) => (
            <Select.Simple<Term> {...p} resourceName="Term">
              <Select.Item itemKey="subscription">Subscription</Select.Item>
              <Select.Item itemKey="perpetual">Perpetual</Select.Item>
            </Select.Simple>
          )}
        </Form.Field>
        <Form.NumericField path="nodes" label="Machines that may activate" />
        <Form.NumericField
          path="channels"
          label="Channel cap per Core"
          inputProps={{ placeholder: "0 for unlimited" }}
        />
        <Form.Field<string>
          path="expiresAt"
          label="Expires (UTC)"
          visible={(_, ctx) => ctx.get<Term>("term")?.value === "subscription"}
        >
          {(p) => <Input.Text {...p} type="date" />}
        </Form.Field>
        <Form.TextField
          path="maxVersion"
          label="Maximum Core version"
          inputProps={{ placeholder: "0.62" }}
        />
        <Text.Text level="small" color={9}>
          Required on a perpetual license. On a subscription it is the version the
          license keeps covering after it expires.
        </Text.Text>
      </Modal.Body>
      <Modal.Footer error={action.error}>
        <Modal.Cancel />
        <Button.Button
          variant="filled"
          onClick={action.run}
          status={action.loading ? "loading" : undefined}
          trigger={["Control", "Enter"]}
          triggerIndicator
        >
          Issue
        </Button.Button>
      </Modal.Footer>
    </Form.Form>
  );
};
