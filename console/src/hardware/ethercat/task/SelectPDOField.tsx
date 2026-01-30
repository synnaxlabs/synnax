// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device as PlutoDevice, Form as PForm, Select } from "@synnaxlabs/pluto";
import { type ReactElement, useMemo } from "react";

import { type PDOEntry, type SlaveProperties } from "@/hardware/ethercat/device/types";

export interface SelectPDOFieldProps {
  path: string;
  pdoType: "inputs" | "outputs";
}

interface PDOOption {
  key: string;
  name: string;
}

export const SelectPDOField = ({
  path,
  pdoType,
}: SelectPDOFieldProps): ReactElement => {
  const slaveKey = PForm.useFieldValue<string>(`${path}.device`);
  const { data: slave } = PlutoDevice.useRetrieve({ key: slaveKey });

  const pdoOptions = useMemo((): PDOOption[] => {
    if (slave == null) return [];
    const props = slave.properties as SlaveProperties | undefined;
    const pdos = props?.pdos?.[pdoType] ?? [];
    return pdos.map((pdo: PDOEntry) => ({
      key: pdo.name,
      name: pdo.name,
    }));
  }, [slave, pdoType]);

  return (
    <PForm.Field<string> path={`${path}.pdo`} label="PDO" grow>
      {({ value, onChange }) => (
        <Select.Static<string, PDOOption>
          value={value}
          onChange={onChange}
          data={pdoOptions}
          resourceName="PDO"
          allowNone={false}
          emptyContent="No PDOs available. Select a slave device first."
        />
      )}
    </PForm.Field>
  );
};
