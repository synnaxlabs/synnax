import { type GLBufferController } from "@synnaxlabs/x";
import { type Mock } from "vitest";
export declare class MockGLBufferController implements GLBufferController {
    ARRAY_BUFFER: number;
    STATIC_DRAW: number;
    DYNAMIC_DRAW: number;
    targets: Record<number, number>;
    counter: number;
    buffers: Record<number, ArrayBuffer>;
    createBufferMock: Mock<() => WebGLBuffer | null>;
    bufferDataMock: Mock<(a: number, b: ArrayBufferLike | number, c: number) => void>;
    bufferSubDataMock: Mock<(a: number, b: number, c: ArrayBufferLike) => void>;
    bindBufferMock: Mock<(a: number, b: WebGLBuffer | null) => void>;
    deleteBufferMock: Mock<(a: WebGLBuffer | null) => void>;
    deleteBuffer(buffer: WebGLBuffer | null): void;
    createBuffer(): WebGLBuffer | null;
    bufferData(target: number, dataOrSize: AllowSharedBufferSource | number, usage: number): void;
    bindBuffer(target: number, buffer: WebGLBuffer | null): void;
    bufferSubData(target: number, offset: number, data: AllowSharedBufferSource): void;
}
//# sourceMappingURL=MockGLBufferController.d.ts.map