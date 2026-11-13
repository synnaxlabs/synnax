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
import { type RefObject, useRef, useState } from "react";

import { list, type ListResult } from "@/feature/sift/asset/list";
import { Select as SelectDevice } from "@/feature/sift/device/Select";
import { Modals } from "@/platform/modals";

export interface DestinationProps extends Omit<
  Form.FieldProps<string>,
  "children" | "label" | "required"
> {
  /** The path of the Sift device whose assets the field lists. */
  devicePath: deep.Target<device.Key>;
}

/**
 * Picks the Sift connection, and an asset from the ones the connection can see. A new
 * asset needs the explicit "New asset" action. The asset starts at the asset of the
 * connection's newest upload, and starts there again when this user picks another
 * connection. A pick by another client of the same form does not move the asset.
 */
export const Destination = ({ devicePath, ...rest }: DestinationProps) => {
  const picked = useRef(false);
  return (
    <>
      <SelectDevice
        onChange={(_, { set }) => {
          picked.current = true;
          set(rest.path, "");
        }}
      />
      <Field {...rest} devicePath={devicePath} picked={picked} />
    </>
  );
};

interface FieldProps extends DestinationProps {
  /** True from this user's pick of a connection until the field lists its assets. */
  picked: RefObject<boolean>;
}

const Field = ({ devicePath, picked, ...rest }: FieldProps) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const deviceKey = Form.useFieldValue(devicePath);
  const { get, set, mode } = Form.useContext();
  const [assets, setAssets] = useState<string[]>([]);
  const listed = useRef(false);
  const rename = Modals.useRename();
  useAsyncEffect(
    async (signal) => {
      setAssets([]);
      if (client == null || deviceKey === "" || mode === "preview") return;
      handleError(async () => {
        let res: ListResult;
        try {
          res = await list(client, deviceKey);
        } catch (e) {
          // A failure after the user moved to another connection or closed the form is
          // stale.
          if (!signal.aborted) handleError(e, "Failed to list Sift assets");
          return;
        }
        if (signal.aborted) return;
        setAssets(res.assets);
        const fill = !listed.current || picked.current;
        listed.current = true;
        picked.current = false;
        if (fill && get<string>(rest.path).value === "") set(rest.path, res.last);
      }, "Failed to list Sift assets");
    },
    [client, deviceKey, mode],
  );
  return (
    <Form.Field<string> {...rest} label="Asset" required>
      {({ value, onChange, preview }) => (
        <Flex.Box x gap="small" grow>
          <Select.Simple<string>
            value={value}
            onChange={onChange}
            preview={preview}
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
          {preview !== true && (
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
          )}
        </Flex.Box>
      )}
    </Form.Field>
  );
};
