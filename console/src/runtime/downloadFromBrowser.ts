// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const downloadFromBrowser = (data: string, type: string, fileName: string) => {
  const link = document.createElement("a");
  link.href = URL.createObjectURL(new Blob([data], { type }));
  link.download = fileName;
  link.click();
  URL.revokeObjectURL(link.href);
  link.remove();
};
