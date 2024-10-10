export class Values {
  // The list of entries in the log, where each entry is a string,
  // and the last entry is the most recent.
  private readonly entries: string[] = [];
  private pos: number = 0;
  private scrollBackMode: boolean = false;

  add(entry: string): void {
    this.entries.push(entry);
  }

  scroll(dist: number) {
    if (!this.scrollBackMode) {
      this.pos = this.entries.length - 1;
      this.scrollBackMode = true;
      return;
    }
    this.pos += dist;
    if (this.pos < 0) this.pos = 0;
    if (this.pos == this.entries.length) this.scrollBackMode = false;
  }

  forView(size: number, cbk: (v: string) => void) {
    if (!this.scrollBackMode) {
      const count = Math.min(size, this.entries.length);
      for (let i = this.entries.length - count; i < this.entries.length; i++)
        cbk(this.entries[i]);
    } else {
      let pos = this.pos;
      if (pos < size) pos = size;
      for (let i = pos; i > Math.max(0, pos - size); i--) cbk(this.entries[i]);
    }
  }
}
