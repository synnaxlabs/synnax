export class BaseTypedError extends Error {
    discriminator = 'FreighterError';
    type;
    constructor(message, type) {
        super(message);
        this.type = type;
    }
}
export const isTypedError = (error) => {
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
export const assertErrorType = (type, error) => {
    if (!error) {
        throw new Error(`Expected error of type ${type} but got nothing instead`);
    }
    if (!isTypedError(error)) {
        throw new Error(`Expected a typed error, got: ${error}`);
    }
    if (error.type !== type) {
        throw new Error(`Expected error of type ${type}, got ${error.type}: ${error}`);
    }
    return error;
};
export const UNKNOWN = 'unknown';
export const NONE = 'nil';
class Registry {
    entries;
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
            return { type: NONE, data: '' };
        }
        if (isTypedError(error) && this.entries[error.type]) {
            return { type: error.type, data: this.entries[error.type].encode(error) };
        }
        return { type: UNKNOWN, data: JSON.stringify(error) };
    }
    decode(payload) {
        if (payload.type === NONE) {
            return undefined;
        }
        if (payload.type === UNKNOWN) {
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
export const registerError = (props) => {
    const { type, ...provider } = props;
    REGISTRY.register(type, provider);
};
export const encodeError = (error) => {
    return REGISTRY.encode(error);
};
export const decodeError = (payload) => {
    return REGISTRY.decode(payload);
};
class UnknownError extends BaseTypedError {
    constructor(message) {
        super(message, UNKNOWN);
    }
}
export class EOF extends BaseTypedError {
    constructor() {
        super('EOF', 'Freighter');
    }
}
export class StreamClosed extends BaseTypedError {
    constructor() {
        super('StreamClosed', 'Freighter');
    }
}
export class Unreachable extends BaseTypedError {
    constructor() {
        super('Unreachable', 'Freighter');
    }
}
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
registerError({
    type: 'freighter',
    encode: freighterErrorEncoder,
    decode: freighterErrorDecoder,
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZXJyb3JzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vc3JjL2xpYi9lcnJvcnMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBYUEsTUFBTSxPQUFPLGNBQWUsU0FBUSxLQUFLO0lBQ3ZDLGFBQWEsR0FBcUIsZ0JBQWdCLENBQUM7SUFDbkQsSUFBSSxDQUFTO0lBRWIsWUFBWSxPQUFlLEVBQUUsSUFBWTtRQUN2QyxLQUFLLENBQUMsT0FBTyxDQUFDLENBQUM7UUFDZixJQUFJLENBQUMsSUFBSSxHQUFHLElBQUksQ0FBQztJQUNuQixDQUFDO0NBQ0Y7QUFLRCxNQUFNLENBQUMsTUFBTSxZQUFZLEdBQUcsQ0FBQyxLQUFjLEVBQXVCLEVBQUU7SUFDbEUsSUFBSSxDQUFDLEtBQUssSUFBSSxPQUFPLEtBQUssS0FBSyxRQUFRLEVBQUU7UUFDdkMsT0FBTyxLQUFLLENBQUM7S0FDZDtJQUNELE1BQU0sVUFBVSxHQUFHLEtBQW1CLENBQUM7SUFDdkMsSUFBSSxVQUFVLENBQUMsYUFBYSxLQUFLLGdCQUFnQixFQUFFO1FBQ2pELE9BQU8sS0FBSyxDQUFDO0tBQ2Q7SUFDRCxJQUFJLENBQUMsQ0FBQyxNQUFNLElBQUksVUFBVSxDQUFDLEVBQUU7UUFDM0IsTUFBTSxJQUFJLEtBQUssQ0FDYixpREFBaUQsVUFBVSxFQUFFLENBQzlELENBQUM7S0FDSDtJQUNELE9BQU8sSUFBSSxDQUFDO0FBQ2QsQ0FBQyxDQUFDO0FBRUYsTUFBTSxDQUFDLE1BQU0sZUFBZSxHQUFHLENBQUksSUFBWSxFQUFFLEtBQWEsRUFBSyxFQUFFO0lBQ25FLElBQUksQ0FBQyxLQUFLLEVBQUU7UUFDVixNQUFNLElBQUksS0FBSyxDQUFDLDBCQUEwQixJQUFJLDBCQUEwQixDQUFDLENBQUM7S0FDM0U7SUFDRCxJQUFJLENBQUMsWUFBWSxDQUFDLEtBQUssQ0FBQyxFQUFFO1FBQ3hCLE1BQU0sSUFBSSxLQUFLLENBQUMsZ0NBQWdDLEtBQUssRUFBRSxDQUFDLENBQUM7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