// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";
import { z } from "zod";

import { type Listed } from "@/server/directory";
import { navigate, post } from "@/ui/api";
import { checkTerms, TermsFields, termsSchema, ZERO_TERMS } from "@/ui/licenses/Terms";
import * as Modal from "@/ui/Modal";

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

const Content = ({ teams }: IssueDialogProps): ReactElement => {
  const methods = Form.use({ values: { ...ZERO_TERMS, organization: "" }, schema });
  return (
    <Modal.Form
      methods={methods}
      submit="Issue"
      onSubmit={async (value) => {
        const { key } = await post<{ key: string }>("/api/licenses", value);
        navigate(`/licenses/${key}`);
      }}
    >
      <Form.Field<string>
        path="organization"
        label="Organization"
        helpText="Organizations and their admins are created in the Clerk dashboard."
      >
        {(p) => (
          <Select.Simple<string> {...p} resourceName="Organization">
            {teams.map(({ clerkOrgID, name }) => (
              <Select.Item key={clerkOrgID} itemKey={clerkOrgID}>
                {name}
              </Select.Item>
            ))}
          </Select.Simple>
        )}
      </Form.Field>
      <TermsFields />
    </Modal.Form>
  );
};
