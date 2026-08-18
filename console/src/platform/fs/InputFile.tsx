// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { type binary } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type z } from "zod";

import { PickerRow, type PickerRowProps } from "@/platform/fs/PickerRow";
import { Runtime } from "@/platform/runtime";

export interface InputFileProps<P extends z.ZodType> extends Omit<
  PickerRowProps,
  "value" | "onClick" | "onChange" | "title"
> {
  /** The displayed name of the loaded file. The caller owns it. */
  value: string;
  /** Receives the decoded contents and the file's name. */
  onChange: (value: z.infer<P>, name: string) => void;
  /** Titles the native dialog. */
  title: string;
  /** Restricts the picker to files with this extension, without the leading dot. */
  extension: string;
  /** Validates the decoded contents. */
  schema: P;
  /** Decodes the file's raw bytes. */
  decoder: binary.Codec;
}

/**
 * A file picker that reads and decodes the chosen file's contents, in the desktop app
 * and the browser alike. Controlled: the caller stores the file's name and receives
 * the decoded contents alongside it. A file that fails to read or decode reports an
 * error status and leaves the value untouched.
 */
export const InputFile = <P extends z.ZodType>({
  value,
  onChange,
  title,
  extension,
  decoder,
  schema,
  ...rest
}: InputFileProps<P>): ReactElement => {
  const handleError = Status.useErrorHandler();
  const handleClick = () =>
    handleError(async () => {
      const file = await Runtime.pickFiles({ title, extension });
      if (file == null) return;
      onChange(decoder.decode<P>(await file.readBytes(), schema), file.path);
    }, "Failed to read file");
  return <PickerRow value={value} onClick={handleClick} {...rest} />;
};
