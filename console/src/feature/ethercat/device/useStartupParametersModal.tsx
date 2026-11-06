// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Text } from "@synnaxlabs/lyra/text";
import { Telem } from "@synnaxlabs/pluto";
import { DataType } from "@synnaxlabs/x";
import { useState } from "react";

import {
  useSlave,
  useUpdateStartupParameters,
} from "@/feature/ethercat/device/queries";
import { formatIndex, type StartupParameter } from "@/feature/ethercat/device/types";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const SUPPORTED_DATA_TYPES = [
  DataType.INT8,
  DataType.INT16,
  DataType.INT32,
  DataType.INT64,
  DataType.UINT8,
  DataType.UINT16,
  DataType.UINT32,
  DataType.UINT64,
  DataType.FLOAT32,
  DataType.FLOAT64,
];

const HIDDEN_DATA_TYPES = DataType.ALL.filter(
  (dt) => !SUPPORTED_DATA_TYPES.some((s) => s.equals(dt)),
);

const HEX_INDEX = /^(0x)?[0-9a-f]{1,4}$/i;

const toRows = (params: StartupParameter[]): Input.TableCell[][] =>
  params.map(({ index, subIndex, dataType, value, name }) => [
    formatIndex(index),
    subIndex,
    dataType,
    value,
    name,
  ]);

const createRow = (): Input.TableCell[] => ["", 0, DataType.UINT8.toString(), 0, ""];

/**
 * Parses the table rows into startup parameters, or returns a message naming the first
 * row whose index is not a hexadecimal number. The Driver checks every other range.
 */
const parseRows = (rows: Input.TableCell[][]): StartupParameter[] | string => {
  const params: StartupParameter[] = [];
  for (const [i, [index, subIndex, dataType, value, name]] of rows.entries()) {
    const text = String(index).trim();
    if (!HEX_INDEX.test(text))
      return `Row ${i + 1}: the index must be a hexadecimal number such as 0x8000.`;
    params.push({
      index: Number.parseInt(text, 16),
      subIndex: Number(subIndex),
      dataType: String(dataType),
      value: Number(value),
      name: String(name),
    });
  }
  return params;
};

interface StartupParametersParams {
  deviceKey: device.Key;
}

const StartupParameters = ({
  deviceKey,
  close,
}: Modals.ContentProps<StartupParametersParams, void>) => {
  const slave = useSlave({ key: deviceKey });
  const [rows, setRows] = useState(() => toRows(slave.properties.startupParameters));
  const [error, setError] = useState<string | null>(null);
  const { update, variant } = useUpdateStartupParameters({
    afterSuccess: () => close(),
  });
  const handleSave = () => {
    const parsed = parseRows(rows);
    if (typeof parsed === "string") return setError(parsed);
    setError(null);
    update({ key: deviceKey, startupParameters: parsed });
  };
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Logo.EtherCAT />}>
        {[slave.name, "Startup parameters"]}
      </Modal.Header>
      <Modal.Body align="stretch" gap="large">
        <Text.Text color={9}>
          The Driver writes these CoE values to the device each time the bus starts.
        </Text.Text>
        <Input.Table value={rows} onChange={setRows} createRow={createRow}>
          <Input.TableColumn name="Index" type="string">
            {(p) => <Input.Text {...p} placeholder="0x8000" />}
          </Input.TableColumn>
          <Input.TableColumn name="Subindex" />
          <Input.TableColumn name="Data type" type="string">
            {({ value, onChange, "aria-label": ariaLabel }) => (
              <Telem.SelectDataType
                value={value}
                onChange={onChange}
                hideDataTypes={HIDDEN_DATA_TYPES}
                triggerProps={{ "aria-label": ariaLabel }}
              />
            )}
          </Input.TableColumn>
          <Input.TableColumn name="Value" />
          <Input.TableColumn name="Name" type="string">
            {(p) => <Input.Text {...p} placeholder="Filter settings" />}
          </Input.TableColumn>
        </Input.Table>
        {error != null && <Text.Text status="error">{error}</Text.Text>}
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End>
          <Button.Button
            status={status.keepVariants(variant, "loading")}
            onClick={handleSave}
            variant="filled"
            trigger={Triggers.SAVE}
            triggerIndicator
            type="submit"
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

export const useStartupParametersModal = Modals.create(StartupParameters);
