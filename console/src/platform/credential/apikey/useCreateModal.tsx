// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/credential/apikey/useCreateModal.css";

import {
  type auth,
  type resource,
  status,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";
import { Credential } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Code } from "@/platform/code";
import {
  DEFAULT_EXPIRY,
  type Expiry,
  resolve,
} from "@/platform/credential/apikey/expiry";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

export interface CreateModalParams {
  /** The user who owns the new key. */
  subject: resource.ID;
}

type CloseProps = Pick<
  Modals.ContentProps<CreateModalParams, auth.apiKey.Key>,
  "close"
>;

interface NameProps extends CloseProps {
  subject: resource.ID;
  onCreated: (created: Credential.APIKey.CreateResult) => void;
}

const Name = ({ subject, close, onCreated }: NameProps): ReactElement => {
  const [name, setName] = useState("");
  const [expiry, setExpiry] = useState<Expiry>(DEFAULT_EXPIRY);
  const [custom, setCustom] = useState(() =>
    Number(TimeStamp.now().add(TimeSpan.days(90)).valueOf()),
  );
  const query = { subject };
  const refreshKeys = Credential.APIKey.useRefresh();
  const { update, variant } = Credential.APIKey.useCreate({
    afterSuccess: ({ data }) => {
      refreshKeys(query);
      onCreated(data);
    },
  });
  const busy = variant === "loading";
  Modals.useBusy(busy);
  const empty = name.trim() === "";
  const customAt = new TimeStamp(custom);
  const now = useNow((now) =>
    expiry === "custom" && customAt.after(now) ? customAt : null,
  );
  const past = expiry === "custom" && !customAt.after(now);
  const handleCreate = (): void => {
    if (empty || past) return;
    const expires = resolve(expiry, customAt, TimeStamp.now());
    update({ subject, name: name.trim(), expires });
  };
  return (
    <>
      <Modal.Body>
        <Input.Text
          level="h2"
          variant="text"
          value={name}
          onChange={setName}
          onKeyDown={(e) => {
            if (e.key === "Enter") handleCreate();
          }}
          placeholder="Key name"
          aria-label="Key name"
          autoFocus
        />
        <Text.Text level="small" color={9}>
          Name the key after the script or service that uses it.
        </Text.Text>
        <Input.Item
          label="Expiry"
          helpText={
            expiry === "never" ? "The key works until you delete it." : undefined
          }
        >
          <Select.Buttons<Expiry> value={expiry} onChange={setExpiry}>
            <Select.Item<Expiry> itemKey="30d">30 days</Select.Item>
            <Select.Item<Expiry> itemKey="90d">90 days</Select.Item>
            <Select.Item<Expiry> itemKey="1y">1 year</Select.Item>
            <Select.Item<Expiry> itemKey="custom">Custom</Select.Item>
            <Select.Item<Expiry> itemKey="never">No expiry</Select.Item>
          </Select.Buttons>
        </Input.Item>
        {expiry === "custom" && (
          <Input.Item
            label="Expiry date"
            status={past ? "error" : undefined}
            helpText={past ? "Expiry must be in the future" : undefined}
          >
            <Input.DateTime
              value={custom}
              onChange={setCustom}
              aria-label="Expiry date"
            />
          </Input.Item>
        )}
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End x align="center">
          <Button.Button
            variant="outlined"
            onClick={() => close()}
            disabled={busy}
            triggerIndicator={PTriggers.ESCAPE}
          >
            Cancel
          </Button.Button>
          <Button.Button
            variant="filled"
            status={status.keepVariants(variant, "loading")}
            disabled={empty || past}
            onClick={handleCreate}
            trigger={Triggers.SAVE}
            triggerIndicator
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </>
  );
};

interface TokenProps extends CloseProps {
  created: Credential.APIKey.CreateResult;
}

const Token = ({ created, close }: TokenProps): ReactElement => (
  <>
    <Modal.Body className={CSS.BE("create-key", "token")}>
      <Text.Text level="h4">{`Copy the key for ${created.key.name}`}</Text.Text>
      <Text.Text color={9}>
        The Core shows it only once. Store it somewhere safe, such as a password
        manager.
      </Text.Text>
      <Code.Snippet label="API key" code={[[created.token]]} primary />
    </Modal.Body>
    <Modal.Footer>
      <Nav.Bar.End x align="center">
        <Button.Button variant="outlined" onClick={() => close(created.key.key)}>
          Done
        </Button.Button>
      </Nav.Bar.End>
    </Modal.Footer>
  </>
);

const Content = ({
  subject,
  close,
}: Modals.ContentProps<CreateModalParams, auth.apiKey.Key>): ReactElement => {
  const [created, setCreated] = useState<Credential.APIKey.CreateResult | null>(null);
  return (
    <Modal.Frame className={CSS.B("create-key")}>
      <Modal.Header icon={<Icon.APIKey />}>Create API key</Modal.Header>
      {created == null ? (
        <Name subject={subject} close={close} onCreated={setCreated} />
      ) : (
        <Token created={created} close={close} />
      )}
    </Modal.Frame>
  );
};

/**
 * Opens a modal that names and creates an API key, then shows its token once. Resolves
 * with the key once the user is done, or null when the user cancels.
 */
export const useCreateModal = Modals.createPrompt<auth.apiKey.Key, CreateModalParams>(
  Content,
);
