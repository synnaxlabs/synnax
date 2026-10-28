// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/license/KeyInput.css";

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import { Runtime } from "@/platform/runtime";

const KEY_FILE_EXTENSION = "lic";

const decoder = new TextDecoder();

/**
 * A field that takes a license key, pasted or loaded from a key file, and activates it
 * on the Core.
 */
export const KeyInput = (): ReactElement => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const [key, setKey] = useState("");
  const [activating, setActivating] = useState(false);
  const [error, setError] = useState<status.Status | null>(null);
  const empty = key.trim() === "";

  const pickFile = (): void =>
    handleError(async () => {
      const file = await Runtime.pickFiles({
        title: "Select a license file",
        extension: KEY_FILE_EXTENSION,
      });
      if (file == null) return;
      const bytes = await Runtime.toBytes(await file.read());
      setKey(decoder.decode(bytes).trim());
    }, "Failed to read license file");

  const activate = async (): Promise<void> => {
    if (client == null || empty) return;
    setActivating(true);
    setError(null);
    try {
      await client.license.activate(key.trim());
      setKey("");
    } catch (e) {
      setError(status.fromException(e, "Failed to activate license"));
    }
    setActivating(false);
  };

  return (
    <Flex.Box y gap="small" full="x">
      <Input.Text
        value={key}
        onChange={setKey}
        placeholder="Paste a license key"
        aria-label="License key"
        full="x"
        disabled={activating}
        onKeyDown={(e) => {
          if (e.key === "Enter") void activate();
        }}
      >
        <Button.Button
          variant="outlined"
          onClick={pickFile}
          disabled={activating}
          tooltip="Load a license file"
          aria-label="Load a license file"
        >
          <Icon.Attachment />
        </Button.Button>
        <Button.Button
          variant="outlined"
          onClick={() => void activate()}
          status={activating ? "loading" : undefined}
          disabled={empty}
        >
          Activate
        </Button.Button>
      </Input.Text>
      <Flex.Box className={CSS.BE("license-key", "status")}>
        {error != null && <Status.Summary status={error} level="small" />}
      </Flex.Box>
    </Flex.Box>
  );
};
