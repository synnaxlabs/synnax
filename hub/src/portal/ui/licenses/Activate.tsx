// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Form, Select, Status, Text } from "@synnaxlabs/pluto";
import { navigate } from "astro:transitions/client";
import { type ReactElement, useCallback, useState } from "react";

import {
  ActivateButton,
  ActivateFields,
  type ActivateSchema,
  useActivate,
} from "@/portal/ui/licenses/ActivateDialog";
import { Empty, Page } from "@/portal/ui/Page";
import { type License, type Organization } from "@/server/db/schema";

export interface ActivateChoice {
  license: License;
  organization: Organization;
}

export interface ActivateProps {
  choices: ActivateChoice[];
  /** selected is the license the page opened on, when the URL named one. */
  selected: string | null;
}

interface Entry {
  key: string;
  name: string;
}

/**
 * Activate is the page the Console links to: pick a license, paste the host hashes,
 * download the token.
 */
export const Activate = ({ choices, selected }: ActivateProps): ReactElement => {
  const entries: Entry[] = choices.map(({ license, organization }) => ({
    key: license.key,
    name: `${license.label || "Untitled license"} (${organization.name})`,
  }));
  const [key, setKey] = useState<string>(selected ?? entries[0]?.key ?? "");
  return (
    <Page title="Activate a machine" subtitle="Give a Core its license token">
      {choices.length === 0 ? (
        <Empty
          message="No license to activate against"
          description="Synnax Labs issues licenses. Contact us to ask for one."
        />
      ) : (
        <Flex.Box
          y
          gap="large"
          bordered
          rounded
          background={1}
          style={{ padding: "4rem" }}
        >
          <Flex.Box y gap="small">
            <Text.Text level="small" color={9}>
              License
            </Text.Text>
            <Select.Static<string, Entry>
              data={entries}
              resourceName="License"
              value={key}
              onChange={setKey}
            />
          </Flex.Box>
          {key !== "" && <Inline key={key} licenseKey={key} />}
        </Flex.Box>
      )}
    </Page>
  );
};

const Inline = ({ licenseKey }: { licenseKey: string }): ReactElement => {
  const { methods, action } = useActivate(
    licenseKey,
    useCallback(async () => {
      await navigate(`/portal/licenses/${licenseKey}`);
    }, [licenseKey]),
  );
  return (
    <Form.Form<ActivateSchema> {...methods}>
      <Flex.Box y gap="medium">
        <ActivateFields />
        <Flex.Box x justify="between" align="center" gap="medium">
          {action.error != null ? (
            <Status.Summary variant="error" level="small" message={action.error} />
          ) : (
            <span />
          )}
          <Flex.Box x gap="small">
            <Button.Button variant="outlined" href="/portal">
              Cancel
            </Button.Button>
            <ActivateButton action={action} />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Form.Form>
  );
};
