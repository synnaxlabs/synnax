export const findParent = (
  el: HTMLElement | null,
  cb: (el: HTMLElement | null) => boolean,
): HTMLElement | null => {
  while (el != null && !cb(el)) el = el.parentElement;
  return el;
};
