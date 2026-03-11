// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm, Select } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useMemo } from "react";

import { useRetrieveSlaveStateful } from "@/hardware/ethercat/device/queries";
import { type PDOEntry } from "@/hardware/ethercat/device/types";

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
  const { data: slave, retrieve } = useRetrieveSlaveStateful();
  useEffect(() => {
    if (primitive.isZero(slaveKey)) return;
    retrieve({ key: slaveKey });
  }, [slaveKey, retrieve]);

  const pdoOptions = useMemo((): PDOOption[] => {
    if (slave == null) return [];
    const pdos = slave.properties?.pdos?.[pdoType] ?? [];
    return pdos.map((pdo: PDOEntry) => ({
      key: pdo.name,
      name: pdo.name,
    }));
  }, [slave, pdoType]);

  const selectRenderProp = useCallback(
    (props: Pick<Select.StaticProps<string, PDOOption>, "value" | "onChange">) => (
      <Select.Static<string, PDOOption>
        {...props}
        data={pdoOptions}
        resourceName="PDO"
        allowNone={false}
        emptyContent="No PDOs available. Select a slave device first."
      />
    ),
    [pdoOptions],
  );

  return (
    <PForm.Field<string> path={`${path}.pdo`} label="PDO" grow>
      {selectRenderProp}
    </PForm.Field>
  );
};
