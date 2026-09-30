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
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { navigate } from "astro:transitions/client";
import { type ReactElement, useState } from "react";

import { type Owned } from "@/server/license/list";
import {
  ActivateButton,
  ActivateFields,
  type ActivateSchema,
  useActivate,
} from "@/ui/licenses/ActivateDialog";
import { Empty, Page } from "@/ui/Page";
import { Panel } from "@/ui/Panel";

export interface ActivateProps {
  choices: Owned[];
  /** selected is the license the page opened on, when the URL named one. */
  selected: string | null;
}

/**
 * Activate is the page the Console links to: pick a license, paste the host hashes,
 * download the license key.
 */
export const Activate = ({ choices, selected }: ActivateProps): ReactElement => {
  const [key, setKey] = useState<string>(selected ?? choices[0]?.license.key ?? "");
  return (
    <Page title="Activate a machine" subtitle="Give a Core its license key">
      {choices.length === 0 ? (
        <Empty
          icon={<Icon.Access />}
          message="No license to activate against"
          description="Synnax Labs issues licenses. Contact us to ask for one."
        />
      ) : (
        <Panel gap="large" className="portal-activate">
          <Flex.Box y gap="small">
            <Text.Text level="small" color={9}>
              License
            </Text.Text>
            <Select.Simple<string> resourceName="License" value={key} onChange={setKey}>
              {choices.map(({ license, organization }) => (
                <Select.Item key={license.key} itemKey={license.key}>
                  {`${license.label} (${organization.name})`}
                </Select.Item>
              ))}
            </Select.Simple>
          </Flex.Box>
          {key !== "" && <Inline licenseKey={key} />}
        </Panel>
      )}
    </Page>
  );
};

const Inline = ({ licenseKey }: { licenseKey: string }): ReactElement => {
  const { methods, action } = useActivate(licenseKey, async () => {
    await navigate(`/licenses/${licenseKey}`);
  });
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
            <Button.Button variant="outlined" href="/">
              Cancel
            </Button.Button>
            <ActivateButton action={action} />
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Form.Form>
  );
};
