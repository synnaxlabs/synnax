// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const urlRegex = new RegExp(
  "^(https?:\\/\\/)?" + // http:// or https:// (optional)
    "((([a-zA-Z0-9][a-zA-Z0-9-]*\\.)+[a-zA-Z]{2,})|" + // domain name and extension
    "localhost|" + // localhost
    "(\\d{1,3}\\.){3}\\d{1,3})" + // or IP address
    "(\\:\\d+)?" + // port (optional)
    "(\\/[-a-zA-Z0-9@:%._\\+~#=]*)*" + // path (optional)
    "(\\?[;&a-zA-Z0-9%_.,~+=-]*)?" + // query string (optional)
    "(#[-a-zA-Z0-9_]*)?$", // fragment identifier (optional)
);

export const is = (string: string): boolean => urlRegex.test(string);
