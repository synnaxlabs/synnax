// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

class TestDataTransfer {
  dropEffect: DataTransfer["dropEffect"] = "none";
  effectAllowed: DataTransfer["effectAllowed"] = "uninitialized";
  readonly files = [] as unknown as FileList;
  readonly items = [] as unknown as DataTransferItemList;
  private readonly data = new Map<string, string>();

  get types(): string[] {
    return [...this.data.keys()];
  }

  getData(format: string): string {
    return this.data.get(format) ?? "";
  }

  setData(format: string, data: string): void {
    this.data.set(format, data);
  }

  clearData(format?: string): void {
    if (format == null) this.data.clear();
    else this.data.delete(format);
  }

  setDragImage(): void {}
}

class TestDragEvent extends MouseEvent {
  readonly dataTransfer: DataTransfer | null;

  constructor(type: string, init?: DragEventInit) {
    super(type, init);
    this.dataTransfer = init?.dataTransfer ?? new TestDataTransfer();
  }
}

/**
 * Installs DragEvent and DataTransfer, which jsdom lacks, so a fired drag event carries
 * a DataTransfer as a browser's does. Its drag image does nothing.
 */
export const installDragEvent = (): void => {
  const globals = { DataTransfer: TestDataTransfer, DragEvent: TestDragEvent };
  Object.assign(globalThis, globals);
  // testing-library builds events from the document's window.
  if (document.defaultView != null) Object.assign(document.defaultView, globals);
};
