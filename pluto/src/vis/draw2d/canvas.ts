// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type Destructor, dimensions, scale, xy } from "@synnaxlabs/x";

import { dimensionsFromMetrics } from "@/text/core/dimensions";
import { applyOverScan } from "@/vis/render/util";
import { type text } from "@/vis/text";

export interface FillTextOptions {
  useAtlas?: boolean;
}

export class SugaredOffscreenCanvasRenderingContext2D
  implements OffscreenCanvasRenderingContext2D
{
  readonly scale_: scale.XY;
  readonly wrapped: OffscreenCanvasRenderingContext2D;
  readonly atlasRegistry: text.AtlasRegistry;
  private cachedFont: string | null = null;
  private cachedFillStyle: string | CanvasGradient | CanvasPattern | null = null;
  private cachedStrokeStyle: string | CanvasGradient | CanvasPattern | null = null;
  private cachedLineWidth: number | null = null;
  private cachedGlobalAlpha: number | null = null;
  private cachedTextAlign: CanvasTextAlign | null = null;
  private cachedTextBaseline: CanvasTextBaseline | null = null;
  private cachedLineCap: CanvasLineCap | null = null;
  private cachedLineJoin: CanvasLineJoin | null = null;
  private cachedMiterLimit: number | null = null;
  private dpr: number;

  constructor(
    wrap: OffscreenCanvasRenderingContext2D,
    atlasRegistry: text.AtlasRegistry,
    dpr: number,
    scale_: scale.XY = scale.XY.IDENTITY,
  ) {
    this.wrapped = wrap;
    this.scale_ = scale_;
    this.atlasRegistry = atlasRegistry;
    this.dpr = dpr;
  }

  get fontStretch(): CanvasFontStretch {
    return this.wrapped.fontStretch;
  }

  set fontStretch(value: CanvasFontStretch) {
    this.wrapped.fontStretch = value;
  }

  get fontVariantCaps(): CanvasFontVariantCaps {
    return this.wrapped.fontVariantCaps;
  }

  set fontVariantCaps(value: CanvasFontVariantCaps) {
    this.wrapped.fontVariantCaps = value;
  }

  isContextLost(): boolean {
    return this.wrapped.isContextLost();
  }

  get wordSpacing(): string {
    return this.wrapped.wordSpacing;
  }

  set wordSpacing(value: string) {
    this.wrapped.wordSpacing = value;
  }

  get letterSpacing(): string {
    return this.wrapped.letterSpacing;
  }

  set letterSpacing(value: string) {
    this.wrapped.letterSpacing = value;
  }

  get textRendering(): CanvasTextRendering {
    return this.wrapped.textRendering;
  }

  set textRendering(value: CanvasTextRendering) {
    this.wrapped.textRendering = value;
  }

  private checkAtlasFillStyle(
    useAtlas: boolean = false,
  ): [true, string] | [false, null] {
    if (useAtlas && typeof this.fillStyle === "string") return [true, this.fillStyle];
    if (useAtlas)
      console.warn(
        "attempted to use a text atlas with a gradient fill style. This is not supported. Falling back to default canvas fill.",
      );
    return [false, null];
  }

  reset(): void {
    this.wrapped.reset();
  }

  applyScale(scale: scale.XY): SugaredOffscreenCanvasRenderingContext2D {
    return new SugaredOffscreenCanvasRenderingContext2D(
      this,
      this.atlasRegistry,
      this.dpr,
      scale,
    );
  }

  get canvas(): OffscreenCanvas {
    return this.wrapped.canvas;
  }

  get miterLimit(): number {
    return this.cachedMiterLimit ?? this.wrapped.miterLimit;
  }

  set miterLimit(value: number) {
    const scaled = this.scale_.x.dim(value);
    if (scaled === this.cachedMiterLimit) return;
    this.cachedMiterLimit = scaled;
    this.wrapped.miterLimit = scaled;
  }

  get globalAlpha(): number {
    return this.cachedGlobalAlpha ?? this.wrapped.globalAlpha;
  }

  set globalAlpha(value: number) {
    if (value === this.cachedGlobalAlpha) return;
    this.cachedGlobalAlpha = value;
    this.wrapped.globalAlpha = value;
  }

  get globalCompositeOperation(): GlobalCompositeOperation {
    return this.wrapped.globalCompositeOperation;
  }

  set globalCompositeOperation(value: GlobalCompositeOperation) {
    this.wrapped.globalCompositeOperation = value;
  }

  get fillStyle(): string | CanvasGradient | CanvasPattern {
    return this.cachedFillStyle ?? this.wrapped.fillStyle;
  }

  set fillStyle(value: string | CanvasGradient | CanvasPattern) {
    if (value === this.cachedFillStyle) return;
    this.cachedFillStyle = value;
    this.wrapped.fillStyle = value;
  }

  get strokeStyle(): string | CanvasGradient | CanvasPattern {
    return this.cachedStrokeStyle ?? this.wrapped.strokeStyle;
  }

  set strokeStyle(value: string | CanvasGradient | CanvasPattern) {
    if (value === this.cachedStrokeStyle) return;
    this.cachedStrokeStyle = value;
    this.wrapped.strokeStyle = value;
  }

  drawImage(image: CanvasImageSource, dx: number, dy: number): void;
  drawImage(
    image: CanvasImageSource,
    dx: number,
    dy: number,
    dw: number,
    dh: number,
  ): void;
  drawImage(
    image: CanvasImageSource,
    sx: number,
    sy: number,
    sw: number,
    sh: number,
    dx: number,
    dy: number,
    dw: number,
    dh: number,
  ): void;
  drawImage(
    image: CanvasImageSource,
    sx: number,
    sy?: number,
    sw?: number,
    sh?: number,
    dx?: number,
    dy?: number,
    dw?: number,
    dh?: number,
  ): void {
    this.wrapped.drawImage(
      image,
      sx,
      sy as number,
      sw as number,
      sh as number,
      (dx != null ? this.scale_.x.pos(dx) : dx) as number,
      (dy != null ? this.scale_.y.pos(dy) : dy) as number,
      (dw != null ? this.scale_.x.dim(dw) : dw) as number,
      (dh != null ? this.scale_.y.dim(dh) : dh) as number,
    );
  }

  beginPath(): void {
    this.wrapped.beginPath();
  }

  clip(fillRule?: CanvasFillRule | undefined): void;
  clip(path: Path2D, fillRule?: CanvasFillRule | undefined): void;
  clip(path?: unknown, fillRule?: unknown): void {
    // @ts-expect-error - typescript overloads cause issues here
    this.wrapped.clip(path, fillRule);
  }

  fill(fillRule?: CanvasFillRule | undefined): void;
  fill(path: Path2D, fillRule?: CanvasFillRule | undefined): void;
  fill(path?: unknown, fillRule?: unknown): void {
    if (path == null) return this.wrapped.fill();
    // @ts-expect-error - typescript overloads cause issues here
    this.wrapped.fill(path, fillRule);
  }

  isPointInPath(x: number, y: number, fillRule?: CanvasFillRule | undefined): boolean;
  isPointInPath(
    path: Path2D,
    x: number,
    y: number,
    fillRule?: CanvasFillRule | undefined,
  ): boolean;
  isPointInPath(path: unknown, x: unknown, y?: unknown, fillRule?: unknown): boolean {
    // @ts-expect-error - typescript overloads cause issues here
    return this.wrapped.isPointInPath(path, x, y, fillRule);
  }

  isPointInStroke(x: number, y: number): boolean;
  isPointInStroke(path: Path2D, x: number, y: number): boolean;
  isPointInStroke(path: unknown, x: unknown, y?: unknown): boolean {
    // @ts-expect-error - typescript overloads cause issues here
    return this.wrapped.isPointInStroke(path, x, y);
  }

  stroke(): void;
  stroke(path: Path2D): void;
  stroke(path?: Path2D): void {
    if (path == null) return this.wrapped.stroke();
    this.wrapped.stroke(path);
  }

  createConicGradient(startAngle: number, x: number, y: number): CanvasGradient {
    return this.wrapped.createConicGradient(startAngle, x, y);
  }

  createLinearGradient(x0: number, y0: number, x1: number, y1: number): CanvasGradient {
    return this.wrapped.createLinearGradient(x0, y0, x1, y1);
  }

  createPattern(
    image: CanvasImageSource,
    repetition: string | null,
  ): CanvasPattern | null {
    return this.wrapped.createPattern(image, repetition);
  }

  createRadialGradient(
    x0: number,
    y0: number,
    r0: number,
    x1: number,
    y1: number,
    r1: number,
  ): CanvasGradient {
    return this.wrapped.createRadialGradient(x0, y0, r0, x1, y1, r1);
  }

  get filter(): string {
    return this.wrapped.filter;
  }

  set filter(value: string) {
    this.wrapped.filter = value;
  }

  createImageData(
    sw: number,
    sh: number,
    settings?: ImageDataSettings | undefined,
  ): ImageData;
  createImageData(imagedata: ImageData): ImageData;
  createImageData(sw: unknown, sh?: unknown, settings?: unknown): ImageData {
    // @ts-expect-error - typescript overloads cause issues here
    return this.wrapped.createImageData(sw, sh, settings);
  }

  getImageData(
    sx: number,
    sy: number,
    sw: number,
    sh: number,
    settings?: ImageDataSettings | undefined,
  ): ImageData {
    return this.wrapped.getImageData(sx, sy, sw, sh, settings);
  }

  putImageData(imagedata: ImageData, dx: number, dy: number): void;
  putImageData(
    imagedata: ImageData,
    dx: number,
    dy: number,
    dirtyX: number,
    dirtyY: number,
    dirtyWidth: number,
    dirtyHeight: number,
  ): void;
  putImageData(
    imagedata: unknown,
    dx: unknown,
    dy: unknown,
    dirtyX?: unknown,
    dirtyY?: unknown,
    dirtyWidth?: unknown,
    dirtyHeight?: unknown,
  ): void {
    this.wrapped.putImageData(
      // @ts-expect-error - typescript overloads cause issues here
      imagedata,
      dx,
      dy,
      dirtyX,
      dirtyY,
      dirtyWidth,
      dirtyHeight,
    );
  }

  get imageSmoothingEnabled(): boolean {
    return this.wrapped.imageSmoothingEnabled;
  }

  set imageSmoothingEnabled(value: boolean) {
    this.wrapped.imageSmoothingEnabled = value;
  }

  get imageSmoothingQuality(): ImageSmoothingQuality {
    return this.wrapped.imageSmoothingQuality;
  }

  set imageSmoothingQuality(value: ImageSmoothingQuality) {
    this.wrapped.imageSmoothingQuality = value;
  }

  arc(
    x: number,
    y: number,
    radius: number,
    startAngle: number,
    endAngle: number,
    counterclockwise?: boolean | undefined,
  ): void {
    this.wrapped.arc(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(radius),
      startAngle,
      endAngle,
      counterclockwise,
    );
  }

  arcTo(x1: number, y1: number, x2: number, y2: number, radius: number): void {
    this.wrapped.arcTo(
      this.scale_.x.pos(x1),
      this.scale_.y.pos(y1),
      this.scale_.x.pos(x2),
      this.scale_.y.pos(y2),
      this.scale_.x.dim(radius),
    );
  }

  bezierCurveTo(
    cp1x: number,
    cp1y: number,
    cp2x: number,
    cp2y: number,
    x: number,
    y: number,
  ): void {
    this.wrapped.bezierCurveTo(
      this.scale_.x.pos(cp1x),
      this.scale_.y.pos(cp1y),
      this.scale_.x.pos(cp2x),
      this.scale_.y.pos(cp2y),
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
    );
  }

  closePath(): void {
    this.wrapped.closePath();
  }

  ellipse(
    x: number,
    y: number,
    radiusX: number,
    radiusY: number,
    rotation: number,
    startAngle: number,
    endAngle: number,
    counterclockwise?: boolean | undefined,
  ): void {
    this.wrapped.ellipse(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(radiusX),
      this.scale_.y.dim(radiusY),
      rotation,
      startAngle,
      endAngle,
      counterclockwise,
    );
  }

  lineTo(x: number, y: number): void {
    this.wrapped.lineTo(this.scale_.x.pos(x), this.scale_.y.pos(y));
  }

  moveTo(x: number, y: number): void {
    this.wrapped.moveTo(this.scale_.x.pos(x), this.scale_.y.pos(y));
  }

  quadraticCurveTo(cpx: number, cpy: number, x: number, y: number): void {
    this.wrapped.quadraticCurveTo(
      this.scale_.x.pos(cpx),
      this.scale_.y.pos(cpy),
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
    );
  }

  rect(x: number, y: number, w: number, h: number): void {
    this.wrapped.rect(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(w),
      this.scale_.y.dim(h),
    );
  }

  roundRect(
    x: number,
    y: number,
    w: number,
    h: number,
    radii?: number | DOMPointInit | Array<number | DOMPointInit> | undefined,
  ): void {
    this.wrapped.roundRect(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(w),
      this.scale_.y.dim(h),
      typeof radii === "number" ? this.scale_.x.dim(radii) : radii,
    );
  }

  get lineCap(): CanvasLineCap {
    return this.cachedLineCap ?? this.wrapped.lineCap;
  }

  set lineCap(value: CanvasLineCap) {
    if (value === this.cachedLineCap) return;
    this.cachedLineCap = value;
    this.wrapped.lineCap = value;
  }

  get lineDashOffset(): number {
    return this.wrapped.lineDashOffset;
  }

  set lineDashOffset(value: number) {
    this.wrapped.lineDashOffset = this.scale_.x.dim(value);
  }

  get lineJoin(): CanvasLineJoin {
    return this.cachedLineJoin ?? this.wrapped.lineJoin;
  }

  set lineJoin(value: CanvasLineJoin) {
    if (value === this.cachedLineJoin) return;
    this.cachedLineJoin = value;
    this.wrapped.lineJoin = value;
  }

  get lineWidth(): number {
    return this.cachedLineWidth ?? this.wrapped.lineWidth;
  }

  set lineWidth(value: number) {
    const scaled = this.scale_.x.dim(value);
    if (scaled === this.cachedLineWidth) return;
    this.cachedLineWidth = scaled;
    this.wrapped.lineWidth = scaled;
  }

  getLineDash(): number[] {
    return this.wrapped.getLineDash();
  }

  setLineDash(segments: number[]): void;
  setLineDash(segments: Iterable<number>): void;
  setLineDash(segments: number[] | Iterable<number>): void {
    const scaled = Array.from(segments).map((v) => this.scale_.x.dim(v));
    this.wrapped.setLineDash(scaled);
  }

  clearRect(x: number, y: number, w: number, h: number): void {
    this.wrapped.clearRect(x, y, w, h);
  }

  fillRect(x: number, y: number, w: number, h: number): void {
    this.wrapped.fillRect(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(w),
      this.scale_.y.dim(h),
    );
  }

  strokeRect(x: number, y: number, w: number, h: number): void {
    this.wrapped.strokeRect(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(w),
      this.scale_.y.dim(h),
    );
  }

  get shadowBlur(): number {
    return this.wrapped.shadowBlur;
  }

  set shadowBlur(value: number) {
    this.wrapped.shadowBlur = value;
  }

  get shadowColor(): string {
    return this.wrapped.shadowColor;
  }

  set shadowColor(value: string) {
    this.wrapped.shadowColor = value;
  }

  get shadowOffsetX(): number {
    return this.wrapped.shadowOffsetX;
  }

  set shadowOffsetX(value: number) {
    this.wrapped.shadowOffsetX = value;
  }

  get shadowOffsetY(): number {
    return this.wrapped.shadowOffsetY;
  }

  restore(): void {
    // Clear all caches on restore since we don't know what state we're restoring to
    this.cachedFillStyle = null;
    this.cachedStrokeStyle = null;
    this.cachedLineWidth = null;
    this.cachedGlobalAlpha = null;
    this.cachedTextAlign = null;
    this.cachedTextBaseline = null;
    this.cachedLineCap = null;
    this.cachedLineJoin = null;
    this.cachedMiterLimit = null;
    this.cachedFont = null;
    this.wrapped.restore();
  }

  save(): void {
    this.wrapped.save();
  }

  fillText(
    text: string,
    x: number,
    y: number,
    maxWidth?: number | undefined,
    options: FillTextOptions = {},
  ): void {
    const [useAtlas, fillStyle] = this.checkAtlasFillStyle(options.useAtlas);
    if (useAtlas) {
      const atlas = this.atlasRegistry.get({ font: this.font, textColor: fillStyle });
      atlas.fillText(this, text, x, y);
      return;
    }
    this.wrapped.fillText(
      text,
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      maxWidth != null ? this.scale_.x.dim(maxWidth) : undefined,
    );
  }

  measureText(text: string): TextMetrics {
    return this.wrapped.measureText(text);
  }

  textDimensions(text: string, options: FillTextOptions = {}): dimensions.Dimensions {
    const [useAtlas, fillStyle] = this.checkAtlasFillStyle(options.useAtlas);
    if (useAtlas) {
      const atlas = this.atlasRegistry.get({ font: this.font, textColor: fillStyle });
      return atlas.measureText(text);
    }
    return dimensionsFromMetrics(this.measureText(text));
  }

  strokeText(text: string, x: number, y: number, maxWidth?: number | undefined): void {
    this.wrapped.strokeText(
      text,
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      maxWidth != null ? this.scale_.x.dim(maxWidth) : undefined,
    );
  }

  get direction(): CanvasDirection {
    return this.wrapped.direction;
  }

  set direction(value: CanvasDirection) {
    this.wrapped.direction = value;
  }

  get font(): string {
    if (this.cachedFont != null) return this.cachedFont;
    return this.wrapped.font;
  }

  set font(value: string) {
    if (value === this.cachedFont) return;
    this.cachedFont = value;
    this.wrapped.font = this.cachedFont;
  }

  get fontKerning(): CanvasFontKerning {
    return this.wrapped.fontKerning;
  }

  set fontKerning(value: CanvasFontKerning) {
    this.wrapped.fontKerning = value;
  }

  get textAlign(): CanvasTextAlign {
    return this.cachedTextAlign ?? this.wrapped.textAlign;
  }

  set textAlign(value: CanvasTextAlign) {
    if (value === this.cachedTextAlign) return;
    this.cachedTextAlign = value;
    this.wrapped.textAlign = value;
  }

  get textBaseline(): CanvasTextBaseline {
    return this.cachedTextBaseline ?? this.wrapped.textBaseline;
  }

  set textBaseline(value: CanvasTextBaseline) {
    if (value === this.cachedTextBaseline) return;
    this.cachedTextBaseline = value;
    this.wrapped.textBaseline = value;
  }

  getTransform(): DOMMatrix {
    return this.wrapped.getTransform();
  }

  resetTransform(): void {
    this.wrapped.resetTransform();
  }

  rotate(angle: number): void {
    this.wrapped.rotate(angle);
  }

  scale(x: number, y: number): void {
    this.wrapped.scale(x, y);
    this.dpr = x;
  }

  scissor(region: box.Box, overScan: xy.XY = xy.ZERO): Destructor {
    const p = new ScaledPath2D(this.scale_);
    region = applyOverScan(region, overScan);
    p.rect(...xy.couple(box.topLeft(region)), ...dimensions.couple(box.dims(region)));
    this.save();
    this.clip(p.getPath());
    return () => this.restore();
  }

  setTransform(a: number, b: number, c: number, d: number, e: number, f: number): void;
  setTransform(transform?: DOMMatrix2DInit | undefined): void;
  setTransform(
    a?: unknown,
    b?: unknown,
    c?: unknown,
    d?: unknown,
    e?: unknown,
    f?: unknown,
  ): void {
    // @ts-expect-error - canvas forwarding
    this.wrapped.setTransform(a, b, c, d, e, f);
  }

  transform(a: number, b: number, c: number, d: number, e: number, f: number): void {
    this.wrapped.transform(a, b, c, d, e, f);
  }

  translate(x: number, y: number): void {
    this.wrapped.translate(x, y);
  }
}

export class ScaledPath2D {
  readonly scale_: scale.XY;
  readonly path: Path2D;

  constructor(scale_: scale.XY = scale.XY.IDENTITY, path?: Path2D | string) {
    this.scale_ = scale_;
    if (path instanceof Path2D || typeof path === "string")
      this.path = new Path2D(path);
    else this.path = new Path2D();
  }

  addPath(path: Path2D, transform?: DOMMatrix2DInit): void {
    this.path.addPath(path, transform);
  }

  arc(
    x: number,
    y: number,
    radius: number,
    startAngle: number,
    endAngle: number,
    counterclockwise?: boolean,
  ): void {
    this.path.arc(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(radius),
      startAngle,
      endAngle,
      counterclockwise,
    );
  }

  arcTo(x1: number, y1: number, x2: number, y2: number, radius: number): void {
    this.path.arcTo(
      this.scale_.x.pos(x1),
      this.scale_.y.pos(y1),
      this.scale_.x.pos(x2),
      this.scale_.y.pos(y2),
      this.scale_.x.dim(radius),
    );
  }

  bezierCurveTo(
    cp1x: number,
    cp1y: number,
    cp2x: number,
    cp2y: number,
    x: number,
    y: number,
  ): void {
    this.path.bezierCurveTo(
      this.scale_.x.pos(cp1x),
      this.scale_.y.pos(cp1y),
      this.scale_.x.pos(cp2x),
      this.scale_.y.pos(cp2y),
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
    );
  }

  closePath(): void {
    this.path.closePath();
  }

  ellipse(
    x: number,
    y: number,
    radiusX: number,
    radiusY: number,
    rotation: number,
    startAngle: number,
    endAngle: number,
    counterclockwise?: boolean,
  ): void {
    this.path.ellipse(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(radiusX),
      this.scale_.y.dim(radiusY),
      rotation,
      startAngle,
      endAngle,
      counterclockwise,
    );
  }

  lineTo(x: number, y: number): void {
    this.path.lineTo(this.scale_.x.pos(x), this.scale_.y.pos(y));
  }

  moveTo(x: number, y: number): void {
    this.path.moveTo(this.scale_.x.pos(x), this.scale_.y.pos(y));
  }

  quadraticCurveTo(cpx: number, cpy: number, x: number, y: number): void {
    this.path.quadraticCurveTo(
      this.scale_.x.pos(cpx),
      this.scale_.y.pos(cpy),
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
    );
  }

  rect(x: number, y: number, w: number, h: number): void {
    this.path.rect(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(w),
      this.scale_.y.dim(h),
    );
  }

  roundRect(
    x: number,
    y: number,
    w: number,
    h: number,
    radii?: number | DOMPointInit | Array<number | DOMPointInit>,
  ): void {
    const scaledRadii = this.scaleRadii(radii);
    this.path.roundRect(
      this.scale_.x.pos(x),
      this.scale_.y.pos(y),
      this.scale_.x.dim(w),
      this.scale_.y.dim(h),
      scaledRadii,
    );
  }

  private scaleRadii(
    radii?: number | DOMPointInit | Array<number | DOMPointInit>,
  ): number | DOMPointInit | Array<number | DOMPointInit> | undefined {
    if (radii == null) return radii;
    if (typeof radii === "number") return this.scale_.x.dim(radii);
    if (Array.isArray(radii)) return radii.map((r) => this.scaleRadius(r));

    return this.scaleRadius(radii);
  }

  private scaleRadius(r: number | DOMPointInit): number | DOMPointInit {
    if (typeof r === "number") return this.scale_.x.dim(r);

    return {
      x: this.scale_.x.dim(r.x ?? 0),
      y: this.scale_.y.dim(r.y ?? 0),
    };
  }

  getPath(): Path2D {
    return this.path;
  }
}
