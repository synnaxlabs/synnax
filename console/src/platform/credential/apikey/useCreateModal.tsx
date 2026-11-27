// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/credential/apikey/useCreateModal.css";

import { type auth, type resource, status, TimeStamp } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";
import { Credential } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Code } from "@/platform/code";
import {
  choices,
  type Expiry,
  preselect,
  preselectCustom,
  reach,
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

const LABELS: Record<Expiry, string> = {
  "30d": "30 days",
  "90d": "90 days",
  "1y": "1 year",
  custom: "Custom",
  never: "No expiry",
};

const Name = ({ subject, close, onCreated }: NameProps): ReactElement => {
  const [name, setName] = useState("");
  const [opened] = useState(() => TimeStamp.now());
  const lifetime = Credential.APIKey.useMaxLifetime({});
  // The choices wait for the limit, so a click never picks one the Core refuses.
  const loaded = lifetime.variant === "success";
  const maxLifetime = loaded ? lifetime.data : null;
  const allowed = choices(maxLifetime);
  const [picked, setExpiry] = useState<Expiry | null>(null);
  const expiry =
    picked != null && allowed.includes(picked) ? picked : preselect(maxLifetime);
  const [pickedCustom, setCustom] = useState<number | null>(null);
  const custom = pickedCustom ?? Number(preselectCustom(opened, maxLifetime).valueOf());
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
  const now = useNow((now) => {
    if (expiry !== "custom" || !customAt.after(now)) return null;
    const span = reach(now, maxLifetime);
    const fitsAt = span == null ? null : customAt.sub(span);
    return fitsAt?.after(now) === true ? fitsAt : customAt;
  });
  const longest = reach(now, maxLifetime);
  const past = expiry === "custom" && !customAt.after(now);
  const tooLate =
    expiry === "custom" && longest != null && customAt.after(now.add(longest));
  const customError = past
    ? "Expiry must be in the future"
    : tooLate
      ? `Expiry must be at most ${longest.toString()} from now`
      : undefined;
  const handleCreate = (): void => {
    if (!loaded || empty || customError != null) return;
    const expires = resolve(expiry, customAt, TimeStamp.now(), maxLifetime);
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
        {lifetime.variant === "error" && (
          <Status.Summary
            variant="error"
            message="Failed to retrieve the key lifetime limit"
            description={lifetime.status.description}
          />
        )}
        {loaded && (
          <Input.Item
            label="Expiry"
            helpText={
              expiry === "never" ? "The key works until you delete it." : undefined
            }
          >
            <Select.Buttons<Expiry> value={expiry} onChange={setExpiry}>
              {allowed.map((e) => (
                <Select.Item<Expiry> key={e} itemKey={e}>
                  {LABELS[e]}
                </Select.Item>
              ))}
            </Select.Buttons>
          </Input.Item>
        )}
        {loaded && expiry === "custom" && (
          <Input.Item
            label="Expiry date"
            status={customError != null ? "error" : undefined}
            helpText={customError}
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
            disabled={!loaded || empty}
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
