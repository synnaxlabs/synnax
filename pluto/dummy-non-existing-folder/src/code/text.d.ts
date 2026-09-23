/** Diff is a single contiguous text change measured in code points: delete deleteCount
 * code points starting at index, then insert insert in their place. */
export interface Diff {
    index: number;
    deleteCount: number;
    insert: string;
}
/** utf16Offset converts a code-point index into the UTF-16 offset a Monaco model
 * addresses with. */
export declare const utf16Offset: (s: string, codePointIndex: number) => number;
/** diff computes the single contiguous change that turns oldStr into newStr, measured in
 * code points: the longest common prefix and suffix are stripped and what remains in the
 * middle is the deletion (from oldStr) and insertion (from newStr). */
export declare const diff: (oldStr: string, newStr: string) => Diff;
//# sourceMappingURL=text.d.ts.map