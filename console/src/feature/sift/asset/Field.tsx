// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { type deep, unique } from "@synnaxlabs/x";
import { useState } from "react";

import { list } from "@/feature/sift/asset/list";
import { Modals } from "@/platform/modals";

export interface FieldProps extends Omit<
  Form.FieldProps<string>,
  "children" | "label" | "required"
> {
  /** The path of the Sift device whose assets the field lists. */
  devicePath: deep.Target<device.Key>;
}

/**
 * Picks a Sift asset from the ones the device can see, or names a new one through an
 * explicit "New asset" action. It starts at the asset of the device's newest upload.
 */
export const Field = ({ devicePath, ...rest }: FieldProps) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const deviceKey = Form.useFieldValue(devicePath);
  const { get, set } = Form.useContext();
  const [assets, setAssets] = useState<string[]>([]);
  const rename = Modals.useRename();
  useAsyncEffect(
    async (signal) => {
      setAssets([]);
      if (client == null || deviceKey === "") return;
      handleError(async () => {
        const res = await list(client, deviceKey);
        if (signal.aborted) return;
        setAssets(res.assets);
        if (get<string>(rest.path).value === "" && res.last !== "")
          set(rest.path, res.last);
      }, "Failed to list Sift assets");
    },
    [client, deviceKey],
  );
  return (
    <Form.Field<string> {...rest} label="Asset" required>
      {({ value, onChange }) => (
        <Flex.Box x gap="small" grow>
          <Select.Simple<string>
            value={value}
            onChange={onChange}
            resourceName="asset"
            emptyContent="No assets in Sift"
            disabled={deviceKey === ""}
            grow
          >
            {unique
              .unique([...assets, value])
              .filter((a) => a !== "")
              .map((a) => (
                <Select.Item key={a} itemKey={a}>
                  {a}
                </Select.Item>
              ))}
          </Select.Simple>
          <Button.Button
            variant="outlined"
            disabled={deviceKey === ""}
            onClick={() =>
              handleError(async () => {
                const name = await rename({
                  title: "New asset",
                  label: "Asset name",
                  icon: <Icon.Logo.Sift />,
                });
                if (name != null) onChange(name);
              }, "Failed to name the new asset")
            }
          >
            <Icon.Add />
            New asset
          </Button.Button>
        </Flex.Box>
      )}
    </Form.Field>
  );
};
