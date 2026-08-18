// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { binary } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type z } from "zod";

import { InputPath, type InputPathProps } from "@/platform/fs/InputPath";
import { Runtime } from "@/platform/runtime";

export interface InputFileProps<P extends z.ZodType = z.ZodString> extends Omit<
  InputPathProps,
  "value" | "onChange"
> {
  /** The path of the currently loaded file. The caller owns it. */
  value: string;
  /** Receives the decoded contents and the path they were read from. */
  onChange: (value: z.infer<P>, path: string) => void;
  schema?: P;
  decoder?: binary.Codec;
}

/**
 * A file picker that reads and decodes the chosen file's contents. Controlled: the
 * caller stores the picked path and receives the decoded contents alongside it. A file
 * that fails to read or decode reports an error status and leaves the value untouched.
 */
export const InputFile = <P extends z.ZodType = z.ZodString>({
  value,
  onChange,
  decoder = binary.TEXT_CODEC,
  schema,
  ...rest
}: InputFileProps<P>): ReactElement => {
  const handleError = Status.useErrorHandler();
  const handleChange = (path: string) =>
    handleError(async () => {
      const contents = await Runtime.readPath(path);
      onChange(decoder.decode<P>(contents, schema), path);
    }, "Failed to read file");
  return <InputPath value={value} onChange={handleChange} {...rest} />;
};
