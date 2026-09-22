// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Dialog, Form, Icon, Select } from "@synnaxlabs/pluto";
import { navigate } from "astro:transitions/client";
import { type ReactElement, useCallback } from "react";
import { z } from "zod";

import { post } from "@/portal/ui/api";
import {
  checkTerms,
  TermsFields,
  termsSchema,
  ZERO_TERMS,
} from "@/portal/ui/licenses/Terms";
import * as Modal from "@/portal/ui/Modal";
import { useAction } from "@/portal/ui/useAction";
import { type Listed } from "@/server/directory";

const schema = termsSchema
  .extend({ organization: z.string().min(1, "Choose an organization") })
  .check(checkTerms);

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
    values: { ...ZERO_TERMS, organization: entries[0]?.key ?? "" },
    schema,
  });
  const action = useAction(
    useCallback(async () => {
      if (!methods.validate()) return;
      const { key } = await post<{ key: string }>("/api/licenses", methods.value());
      await navigate(`/account/licenses/${key}`);
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
            <Select.Static<string, OrganizationEntry>
              {...p}
              data={entries}
              resourceName="Organization"
            />
          )}
        </Form.Field>
        <TermsFields />
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
