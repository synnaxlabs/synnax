// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/credential/useCreateKeyModal.css";

import { type auth, type resource, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";
import { Credential } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Code } from "@/platform/code";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

export interface CreateKeyModalParams {
  /** The user who owns the new key. */
  subject: resource.ID;
}

type CloseProps = Pick<
  Modals.ContentProps<CreateKeyModalParams, auth.APIKeyKey>,
  "close"
>;

interface NameProps extends CloseProps {
  subject: resource.ID;
  onCreated: (created: Credential.CreateKeyResult) => void;
}

const Name = ({ subject, close, onCreated }: NameProps): ReactElement => {
  const [name, setName] = useState("");
  const query = { subject };
  const refreshKeys = Credential.useRefreshKeys();
  const { update, variant } = Credential.useCreateKey({
    afterSuccess: ({ data }) => {
      refreshKeys(query);
      onCreated(data);
    },
  });
  const busy = variant === "loading";
  Modals.useBusy(busy);
  const empty = name.trim() === "";
  const handleCreate = (): void => {
    if (!empty) update({ subject, name: name.trim() });
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
            disabled={empty}
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
  created: Credential.CreateKeyResult;
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
}: Modals.ContentProps<CreateKeyModalParams, auth.APIKeyKey>): ReactElement => {
  const [created, setCreated] = useState<Credential.CreateKeyResult | null>(null);
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
export const useCreateKeyModal = Modals.createPrompt<
  auth.APIKeyKey,
  CreateKeyModalParams
>(Content);
