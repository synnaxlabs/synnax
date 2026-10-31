// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ethercat } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type z } from "zod";

import { Device } from "@/feature/ethercat/device";
import {
  addressKey,
  type Channel,
  DIRECTIONS,
  type EntryChannel,
  getEntryLabel,
  getUnresolvedMessage,
  hasEntry,
  type PDOType,
  uniqueEntries,
} from "@/feature/ethercat/task/types";

export interface SelectEntryFieldProps {
  path: string;
  pdoType: PDOType;
  schema: z.ZodType<EntryChannel>;
}

export const SelectEntryField = ({
  path,
  pdoType,
  schema,
}: SelectEntryFieldProps): ReactElement => {
  const ch = PForm.useFieldValue<Channel>(path);
  const { data: slave } = Device.useResultSlave(
    primitive.isZero(ch.device) ? null : { key: ch.device },
  );
  const entries = uniqueEntries(slave?.properties.pdos[pdoType] ?? []);
  const emptyContent = primitive.isZero(ch.device)
    ? "Select a slave device first."
    : `This slave has no ${DIRECTIONS[pdoType]} entries. Scan the network again.`;
  const unresolvedProps =
    ch.type === "unresolved"
      ? {
          status: "error" as const,
          helpText:
            slave == null ? undefined : getUnresolvedMessage(ch.pdo, slave, pdoType),
        }
      : {};
  return (
    <PForm.Field<Channel> path={path} label="Entry" grow {...unresolvedProps}>
      {({ value, onChange }) => (
        <Select.Simple<string>
          value={hasEntry(value) ? addressKey(value) : ""}
          onChange={(key: string) => {
            const entry = entries.find((e) => addressKey(e) === key);
            if (entry == null) throw new Error(`No entry has the key ${key}`);
            const { index, subIndex, dataType } = entry;
            onChange(
              schema.parse({ ...value, type: "entry", index, subIndex, dataType }),
            );
          }}
          resourceName="entry"
          allowNone={false}
          emptyContent={emptyContent}
        >
          {entries.map((entry) => {
            const label = getEntryLabel(entry);
            const address = ethercat.formatAddress(entry);
            return (
              <Select.Item
                key={addressKey(entry)}
                itemKey={addressKey(entry)}
                textValue={`${label} ${address}`}
                justify="between"
              >
                <Text.Text>{label}</Text.Text>
                <Text.Text level="small" color={9}>
                  {address}
                </Text.Text>
              </Select.Item>
            );
          })}
        </Select.Simple>
      )}
    </PForm.Field>
  );
};
