"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.camelKeys = exports.snakeKeys = void 0;
const js_convert_case_1 = require("js-convert-case");
const options = {
    recursive: true,
    recursiveInArray: true,
};
const snakeKeys = (entity) => (0, js_convert_case_1.snakeKeys)(entity, options);
exports.snakeKeys = snakeKeys;
const camelKeys = (entity) => (0, js_convert_case_1.camelKeys)(entity, options);
exports.camelKeys = camelKeys;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY2FzZWNvbnYuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi8uLi9zcmMvbGliL2Nhc2Vjb252LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiI7OztBQUFBLHFEQUd5QjtBQUV6QixNQUFNLE9BQU8sR0FBRztJQUNkLFNBQVMsRUFBRSxJQUFJO0lBQ2YsZ0JBQWdCLEVBQUUsSUFBSTtDQUN2QixDQUFDO0FBRUssTUFBTSxTQUFTLEdBQUcsQ0FBQyxNQUFlLEVBQUUsRUFBRSxDQUFDLElBQUEsMkJBQVUsRUFBQyxNQUFNLEVBQUUsT0FBTyxDQUFDLENBQUM7QUFBN0QsUUFBQSxTQUFTLGFBQW9EO0FBQ25FLE1BQU0sU0FBUyxHQUFHLENBQUMsTUFBZSxFQUFFLEVBQUUsQ0FBQyxJQUFBLDJCQUFVLEVBQUMsTUFBTSxFQUFFLE9BQU8sQ0FBQyxDQUFDO0FBQTdELFFBQUEsU0FBUyxhQUFvRCJ9