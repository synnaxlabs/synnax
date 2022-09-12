"use strict";
var __rest = (this && this.__rest) || function (s, e) {
    var t = {};
    for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p) && e.indexOf(p) < 0)
        t[p] = s[p];
    if (s != null && typeof Object.getOwnPropertySymbols === "function")
        for (var i = 0, p = Object.getOwnPropertySymbols(s); i < p.length; i++) {
            if (e.indexOf(p[i]) < 0 && Object.prototype.propertyIsEnumerable.call(s, p[i]))
                t[p[i]] = s[p[i]];
        }
    return t;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Unreachable = exports.StreamClosed = exports.EOF = exports.decodeError = exports.encodeError = exports.registerError = exports.NONE = exports.UNKNOWN = exports.assertErrorType = exports.isTypedError = exports.BaseTypedError = void 0;
class BaseTypedError extends Error {
    constructor(message, type) {
        super(message);
        this.discriminator = 'FreighterError';
        this.type = type;
    }
}
exports.BaseTypedError = BaseTypedError;
const isTypedError = (error) => {
    if (!error || typeof error !== 'object') {
        return false;
    }
    const typedError = error;
    if (typedError.discriminator !== 'FreighterError') {
        return false;
    }
    if (!('type' in typedError)) {
        throw new Error(`Freighter error is missing its type property: ${typedError}`);
    }
    return true;
};
exports.isTypedError = isTypedError;
const assertErrorType = (type, error) => {
    if (!error) {
        throw new Error(`Expected error of type ${type} but got nothing instead`);
    }
    if (!(0, exports.isTypedError)(error)) {
        throw new Error(`Expected a typed error, got: ${error}`);
    }
    if (error.type !== type) {
        throw new Error(`Expected error of type ${type}, got ${error.type}: ${error}`);
    }
    return error;
};
exports.assertErrorType = assertErrorType;
exports.UNKNOWN = 'unknown';
exports.NONE = 'nil';
class Registry {
    constructor() {
        this.entries = {};
    }
    register(_type, provider) {
        if (this.entries[_type]) {
            throw new Error(`Error type ${_type} is already registered`);
        }
        this.entries[_type] = provider;
    }
    encode(error) {
        if (!error) {
            return { type: exports.NONE, data: '' };
        }
        if ((0, exports.isTypedError)(error) && this.entries[error.type]) {
            return { type: error.type, data: this.entries[error.type].encode(error) };
        }
        return { type: exports.UNKNOWN, data: JSON.stringify(error) };
    }
    decode(payload) {
        if (payload.type === exports.NONE) {
            return undefined;
        }
        if (payload.type === exports.UNKNOWN) {
            return new UnknownError(payload.data);
        }
        const provider = this.entries[payload.type];
        if (!provider) {
            return new UnknownError(payload.data);
        }
        return provider.decode(payload.data);
    }
}
const REGISTRY = new Registry();
const registerError = (props) => {
    const { type } = props, provider = __rest(props, ["type"]);
    REGISTRY.register(type, provider);
};
exports.registerError = registerError;
const encodeError = (error) => {
    return REGISTRY.encode(error);
};
exports.encodeError = encodeError;
const decodeError = (payload) => {
    return REGISTRY.decode(payload);
};
exports.decodeError = decodeError;
class UnknownError extends BaseTypedError {
    constructor(message) {
        super(message, exports.UNKNOWN);
    }
}
class EOF extends BaseTypedError {
    constructor() {
        super('EOF', 'Freighter');
    }
}
exports.EOF = EOF;
class StreamClosed extends BaseTypedError {
    constructor() {
        super('StreamClosed', 'Freighter');
    }
}
exports.StreamClosed = StreamClosed;
class Unreachable extends BaseTypedError {
    constructor() {
        super('Unreachable', 'Freighter');
    }
}
exports.Unreachable = Unreachable;
const freighterErrorEncoder = (error) => {
    if (error instanceof EOF) {
        return 'EOF';
    }
    if (error instanceof StreamClosed) {
        return 'StreamClosed';
    }
    if (error instanceof Unreachable) {
        return 'Unreachable';
    }
    throw new Error(`Unknown error type: ${error}`);
};
const freighterErrorDecoder = (encoded) => {
    switch (encoded) {
        case 'EOF':
            return new EOF();
        case 'StreamClosed':
            return new StreamClosed();
        case 'Unreachable':
            return new Unreachable();
        default:
            throw new Error(`Unknown error type: ${encoded}`);
    }
};
(0, exports.registerError)({
    type: 'freighter',
    encode: freighterErrorEncoder,
    decode: freighterErrorDecoder,
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZXJyb3JzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vc3JjL2xpYi9lcnJvcnMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6Ijs7Ozs7Ozs7Ozs7Ozs7QUFhQSxNQUFhLGNBQWUsU0FBUSxLQUFLO0lBSXZDLFlBQVksT0FBZSxFQUFFLElBQVk7UUFDdkMsS0FBSyxDQUFDLE9BQU8sQ0FBQyxDQUFDO1FBSmpCLGtCQUFhLEdBQXFCLGdCQUFnQixDQUFDO1FBS2pELElBQUksQ0FBQyxJQUFJLEdBQUcsSUFBSSxDQUFDO0lBQ25CLENBQUM7Q0FDRjtBQVJELHdDQVFDO0FBS00sTUFBTSxZQUFZLEdBQUcsQ0FBQyxLQUFjLEVBQXVCLEVBQUU7SUFDbEUsSUFBSSxDQUFDLEtBQUssSUFBSSxPQUFPLEtBQUssS0FBSyxRQUFRLEVBQUU7UUFDdkMsT0FBTyxLQUFLLENBQUM7S0FDZDtJQUNELE1BQU0sVUFBVSxHQUFHLEtBQW1CLENBQUM7SUFDdkMsSUFBSSxVQUFVLENBQUMsYUFBYSxLQUFLLGdCQUFnQixFQUFFO1FBQ2pELE9BQU8sS0FBSyxDQUFDO0tBQ2Q7SUFDRCxJQUFJLENBQUMsQ0FBQyxNQUFNLElBQUksVUFBVSxDQUFDLEVBQUU7UUFDM0IsTUFBTSxJQUFJLEtBQUssQ0FDYixpREFBaUQsVUFBVSxFQUFFLENBQzlELENBQUM7S0FDSDtJQUNELE9BQU8sSUFBSSxDQUFDO0FBQ2QsQ0FBQyxDQUFDO0FBZFcsUUFBQSxZQUFZLGdCQWN2QjtBQUVLLE1BQU0sZUFBZSxHQUFHLENBQUksSUFBWSxFQUFFLEtBQWEsRUFBSyxFQUFFO0lBQ25FLElBQUksQ0FBQyxLQUFLLEVBQUU7UUFDVixNQUFNLElBQUksS0FBSyxDQUFDLDBCQUEwQixJQUFJLDBCQUEwQixDQUFDLENBQUM7S0FDM0U7SUFDRCxJQUFJLENBQUMsSUFBQSxvQkFBWSxFQUFDLEtBQUssQ0FBQyxFQUFFO1FBQ3hCLE1BQU0sSUFBSSxLQUFLLENBQUMsZ0NBQWdDLEtBQUss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