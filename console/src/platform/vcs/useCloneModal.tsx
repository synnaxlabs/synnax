// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useState } from "react";

import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";
import { useSetStatus } from "@/platform/vcs/Provider";

export interface CloneModalParams {
  /** Says where the new configuration starts, under the name field. */
  description?: string;
}

const NAME_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._-]*$/;

const validate = (name: string): string | undefined => {
  if (name.length === 0) return "Name is required";
  if (!NAME_PATTERN.test(name))
    return "Use letters, numbers, dots, dashes, and underscores";
  return undefined;
};

const Clone = ({
  description,
  close,
}: Modals.ContentProps<CloneModalParams, string>) => {
  const client = Synnax.use();
  const setStatus = useSetStatus();
  const addStatus = Status.useAdder();
  const [name, setName] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [loading, setLoading] = useState(false);
  const handleCreate = async () => {
    const invalid = validate(name);
    if (invalid != null) return setError(invalid);
    if (client == null) return;
    setLoading(true);
    try {
      setStatus(await client.vcs.clone(name));
      addStatus({ variant: "success", name, message: "Created configuration" });
      close(name);
    } catch (e) {
      setError(errors.fromUnknown(e).message);
    }
    setLoading(false);
  };
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Branch />}>New configuration</Modal.Header>
      <Modal.Body>
        <Input.Item
          showLabel={false}
          helpText={error}
          status={error != null ? "error" : undefined}
          padHelpText
        >
          <Input.Text
            autoFocus
            placeholder="Configuration name"
            aria-label="Configuration name"
            level="h3"
            variant="text"
            value={name}
            onChange={(v) => {
              // Configurations are Git branches, which cannot hold spaces.
              setName(v.replace(/\s/g, "-"));
              setError(undefined);
            }}
          />
        </Input.Item>
        <Text.Text level="small" color={9}>
          {description ??
            "The new configuration starts from the last commit. Nothing on the Core changes, and uncommitted changes stay."}
        </Text.Text>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End x align="center">
          <Button.Button
            variant="filled"
            status={loading ? "loading" : undefined}
            disabled={client == null || name.length === 0}
            onClick={() => void handleCreate()}
            trigger={Triggers.SAVE}
            triggerIndicator
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/**
 * Opens a dialog that creates a configuration from the current one. Resolves with its
 * name, or null when dismissed.
 */
export const useCloneModal = Modals.createPrompt<string, CloneModalParams>(Clone);
