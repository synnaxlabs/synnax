"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.newClient = exports.PORT = exports.HOST = void 0;
const client_1 = __importDefault(require("./lib/client"));
exports.HOST = 'localhost';
exports.PORT = 8080;
const newClient = (...props) => {
    let _props = {};
    if (props.length > 0)
        _props = props[0];
    return new client_1.default(Object.assign({ host: exports.HOST, port: exports.PORT, username: 'synnax', password: 'seldon' }, _props));
};
exports.newClient = newClient;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoic2V0dXBzcGVjcy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uL3NyYy9zZXR1cHNwZWNzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiI7Ozs7OztBQUFBLDBEQUFtRDtBQUV0QyxRQUFBLElBQUksR0FBRyxXQUFXLENBQUM7QUFDbkIsUUFBQSxJQUFJLEdBQUcsSUFBSSxDQUFDO0FBRWxCLE1BQU0sU0FBUyxHQUFHLENBQUMsR0FBRyxLQUFvQixFQUFVLEVBQUU7SUFDM0QsSUFBSSxNQUFNLEdBQUcsRUFBRSxDQUFDO0lBQ2hCLElBQUksS0FBSyxDQUFDLE1BQU0sR0FBRyxDQUFDO1FBQUUsTUFBTSxHQUFHLEtBQUssQ0FBQyxDQUFDLENBQUMsQ0FBQztJQUN4QyxPQUFPLElBQUksZ0JBQU0saUJBQ2YsSUFBSSxFQUFFLFlBQUksRUFDVixJQUFJLEVBQUUsWUFBSSxFQUNWLFFBQVEsRUFBRSxRQUFRLEVBQ2xCLFFBQVEsRUFBRSxRQUFRLElBQ2YsTUFBTSxFQUNULENBQUM7QUFDTCxDQUFDLENBQUM7QUFWVyxRQUFBLFNBQVMsYUFVcEIifQ==