import { transform } from "framer-motion";

export type XYTuple = [number, number];

export const addXYTuples = (a: XYTuple, b: XYTuple): XYTuple => {
  return [a[0] + b[0], a[1] + b[1]];
};

export const multiplyXYTuples = (a: XYTuple, b: XYTuple): XYTuple => {
  return [a[0] * b[0], a[1] * b[1]];
};

export type XYRect = {
  position: XYTuple;
  dimensions: XYTuple;
};

export type XYTransform = {
  scale: XYTuple;
  translation: XYTuple;
};

export const normalToClipSpace: XYTransform = {
  scale: [2, 2],
  translation: [-1, -1],
};

export const applyTransform = (
  transform: XYTransform,
  point: XYTuple
): XYTuple => {
  return [
    transform.scale[0] * point[0] + transform.translation[0],
    transform.scale[1] * point[1] + transform.translation[1],
  ];
};

export const pixelsToNormal = (bounds: XYTuple): XYTransform => {
  return {
    scale: [1 / bounds[0], -1 / bounds[1]],
    translation: [0, 1],
  };
};

export const _combineTransforms = (
  a: XYTransform,
  b: XYTransform
): XYTransform => {
  const scale: XYTuple = [a.scale[0] * b.scale[0], a.scale[1] * b.scale[1]];
  const translation: XYTuple = [
    a.translation[0] * b.scale[0] + b.translation[0],
    a.translation[1] * b.scale[1] + b.translation[1],
  ];
  return { scale, translation };
};

export const combineTransforms = (
  ...transforms: XYTransform[]
): XYTransform => {
  return transforms.reduce(_combineTransforms);
};

export const pixelsToClipSpace = (bounds: XYTuple): XYTransform => {
  return combineTransforms(pixelsToNormal(bounds), normalToClipSpace);
};
