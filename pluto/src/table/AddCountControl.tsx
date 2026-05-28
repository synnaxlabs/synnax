// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import { Button } from "@/button";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Input } from "@/input";

export interface AddCountControlProps {
  resourceName: "row" | "column";
  onAdd: (count: number) => void;
  className?: string;
}

export const AddCountControl = ({
  resourceName,
  onAdd,
  className,
}: AddCountControlProps): ReactElement => {
  const [count, setCount] = useState(1);
  return (
    <Flex.Box className={className} x pack align="center" justify="center">
      <Input.Numeric
        value={count}
        onChange={setCount}
        bounds={{ lower: 1, upper: 100 }}
        size="tiny"
        showDragHandle={false}
      />
      <Button.Button
        size="tiny"
        variant="filled"
        onClick={() => {
          onAdd(count);
          setCount(1);
        }}
        tooltip={`Add ${count} ${count === 1 ? resourceName : `${resourceName}s`}`}
      >
        <Icon.Add />
      </Button.Button>
    </Flex.Box>
  );
};
