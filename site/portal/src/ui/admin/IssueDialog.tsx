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
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";
import { z } from "zod";

import { type Listed } from "@/server/directory";
import { navigate, post } from "@/ui/api";
import { Licenses } from "@/ui/licenses";
import { Modal } from "@/ui/modal";

const schema = Licenses.termsSchema
  .extend({ organization: z.string().min(1, "Choose an organization") })
  .check(Licenses.checkTerms);

export interface IssueDialogProps {
  /** teams are the organizations to choose from. With one, the dialog issues to it. */
  teams: Listed[];
}

/** IssueDialog issues a new enterprise license to an organization. Staff only. */
export const IssueDialog = ({ teams }: IssueDialogProps): ReactElement => (
  <Modal.Frame
    name={teams.length === 1 ? `${teams[0].name}.Issue a license` : "Issue a license"}
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
  const methods = Form.use({
    values: {
      ...Licenses.ZERO_TERMS,
      organization: teams.length === 1 ? teams[0].clerkOrgID : "",
    },
    schema,
  });
  return (
    <Modal.Form
      methods={methods}
      submit="Issue"
      onSubmit={async (value) => {
        const { key } = await post<{ key: string }>(
          "/api/licenses",
          Licenses.bodyOf(value),
        );
        navigate(`/licenses/${key}`);
      }}
    >
      <Text.Text level="p" color={10}>
        The team's admins get an email with its terms and a link to it.
      </Text.Text>
      {teams.length !== 1 && (
        <Form.Field<string>
          path="organization"
          label="Organization"
          helpText="Register an organization under Organizations first."
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
      )}
      <Licenses.TermsFields />
    </Modal.Form>
  );
};
