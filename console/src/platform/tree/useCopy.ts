// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clipboard } from "@synnaxlabs/lyra/clipboard";
import { Status } from "@synnaxlabs/lyra/status";
import { strings } from "@synnaxlabs/x";

import { Link } from "@/platform/link";
import { useItems } from "@/platform/tree/Provider";
import { type Resource } from "@/platform/tree/types";
import { Session } from "@/session";

const write = async (text: string, html: string): Promise<void> => {
  // An insecure (http) context has no ClipboardItem.
  if (typeof ClipboardItem !== "undefined")
    try {
      await navigator.clipboard.write([
        new ClipboardItem({
          "text/plain": new Blob([text], { type: "text/plain" }),
          "text/html": new Blob([html], { type: "text/html" }),
        }),
      ]);
      return;
    } catch {
      // The plain text still copies where the rich write is refused.
    }
  await clipboard.writeText(text);
};

export interface Copy {
  /** Writes the names into the data of a native copy event. */
  intoEvent: (resources: Resource[], data: DataTransfer) => void;
  /** Writes the names to the clipboard. */
  toClipboard: (resources: Resource[]) => void;
}

/**
 * Copies tree resources in two forms. Plain text holds their names, one per line, for
 * code. HTML holds the same names as links where a link opens the type, for chat and
 * email.
 */
export const useCopy = (): Copy => {
  const items = useItems();
  const clusterKey = Session.Core.useSelectClusterKey();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();

  const format = (resources: Resource[]) => {
    const text = resources.map(({ name }) => name).join("\n");
    const html = resources
      .map(({ id, name }) => {
        const escaped = strings.escapeHTML(name);
        if (
          Session.Runtime.LINKS_DISABLED ||
          clusterKey == null ||
          items[id.type]?.linkable !== true
        )
          return escaped;
        const href = strings.escapeHTML(Link.create(clusterKey, id));
        return `<a href="${href}">${escaped}</a>`;
      })
      .join("<br>");
    const single = resources.length === 1;
    const name = single ? resources[0].name : "";
    const what = single ? "name" : `${resources.length} names`;
    return { text, html, name, what };
  };

  const intoEvent = (resources: Resource[], data: DataTransfer) => {
    const { text, html, name, what } = format(resources);
    data.setData("text/plain", text);
    data.setData("text/html", html);
    addStatus({ variant: "success", name, message: `Copied ${what} to clipboard` });
  };

  const toClipboard = (resources: Resource[]) => {
    const { text, html, name, what } = format(resources);
    handleError(
      async () => {
        await write(text, html);
        addStatus({ variant: "success", name, message: `Copied ${what} to clipboard` });
      },
      { message: `Failed to copy ${what} to clipboard`, name },
    );
  };

  return { intoEvent, toClipboard };
};
