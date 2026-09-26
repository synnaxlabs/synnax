// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";

import { useResultSlave } from "@/feature/ethercat/device/queries";
import { type PDOEntry } from "@/feature/ethercat/device/types";

export interface SelectPDOFieldProps {
  path: string;
  pdoType: "inputs" | "outputs";
}

export const SelectPDOField = ({
  path,
  pdoType,
}: SelectPDOFieldProps): ReactElement => {
  const slaveKey = PForm.useFieldValue<string>(`${path}.device`);
  const { data: slave } = useResultSlave(
    primitive.isZero(slaveKey) ? null : { key: slaveKey },
  );

  const pdos = useMemo(
    (): PDOEntry[] => slave?.properties?.pdos?.[pdoType] ?? [],
    [slave, pdoType],
  );

  const selectRenderProp = useCallback(
    (props: Pick<Select.SimpleProps<string>, "value" | "onChange">) => (
      <Select.Simple<string>
        {...props}
        resourceName="PDO"
        allowNone={false}
        emptyContent="No PDOs available. Select a slave device first."
      >
        {pdos.map(({ name }) => (
          <Select.Item key={name} itemKey={name}>
            {name}
          </Select.Item>
        ))}
      </Select.Simple>
    ),
    [pdos],
  );

  return (
    <PForm.Field<string> path={`${path}.pdo`} label="PDO" grow>
      {selectRenderProp}
    </PForm.Field>
  );
};
