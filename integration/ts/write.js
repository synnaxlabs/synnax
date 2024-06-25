"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (g && (g = 0, op[0] && (_ = 0)), _) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
var __spreadArray = (this && this.__spreadArray) || function (to, from, pack) {
    if (pack || arguments.length === 2) for (var i = 0, l = from.length, ar; i < l; i++) {
        if (ar || !(i in from)) {
            if (!ar) ar = Array.prototype.slice.call(from, 0, i);
            ar[i] = from[i];
        }
    }
    return to.concat(ar || Array.prototype.slice.call(from));
};
Object.defineProperty(exports, "__esModule", { value: true });
var x_1 = require("@synnaxlabs/x");
var client_1 = require("@synnaxlabs/client");
var process_1 = require("process");
var client = new client_1.Synnax({
    host: "localhost",
    port: 9090,
    username: "synnax",
    password: "seldon",
    secure: false,
});
var IndexWriterGroup = /** @class */ (function () {
    function IndexWriterGroup(indexChannels, dataChannels) {
        this.indexChannels = [];
        this.dataChannels = [];
        this.indexChannels = indexChannels;
        this.dataChannels = dataChannels;
    }
    IndexWriterGroup.prototype.together = function () {
        return __spreadArray(__spreadArray([], this.indexChannels, true), this.dataChannels, true);
    };
    return IndexWriterGroup;
}());
var TestConfig = /** @class */ (function () {
    function TestConfig() {
        this.identifier = "";
        this.numWriters = 0;
        this.domains = 0;
        this.samplesPerDomain = 0;
        this.timeRange = x_1.TimeRange.ZERO;
        this.autoCommit = false;
        this.indexPersistInterval = x_1.TimeSpan.seconds(1);
        this.writerMode = client_1.framer.WriterMode.PersistStream;
        this.channels = [];
    }
    return TestConfig;
}());
function writeTest(tc) {
    return __awaiter(this, void 0, void 0, function () {
        var writers, timeSpanPerDomain, i, _a, _b, tsHwm_1, _loop_1, i, _i, writers_1, writer;
        return __generator(this, function (_c) {
            switch (_c.label) {
                case 0:
                    writers = new Array(tc.numWriters).fill(null);
                    timeSpanPerDomain = Number(tc.timeRange.span) / tc.domains;
                    console.log(tc.channels);
                    i = 0;
                    _c.label = 1;
                case 1:
                    if (!(i < tc.numWriters)) return [3 /*break*/, 4];
                    _a = writers;
                    _b = i;
                    return [4 /*yield*/, client.openWriter({
                            start: tc.timeRange.start,
                            channels: tc.channels[i].together(),
                            mode: tc.writerMode,
                            enableAutoCommit: tc.autoCommit,
                            autoIndexPersistInterval: tc.indexPersistInterval,
                        })];
                case 2:
                    _a[_b] = _c.sent();
                    _c.label = 3;
                case 3:
                    i++;
                    return [3 /*break*/, 1];
                case 4:
                    _c.trys.push([4, , 9, 14]);
                    tsHwm_1 = tc.timeRange.start.add(new x_1.TimeSpan(1));
                    _loop_1 = function (i) {
                        var timestamps, data, _loop_2, j;
                        return __generator(this, function (_d) {
                            switch (_d.label) {
                                case 0:
                                    timestamps = Array.from({ length: tc.samplesPerDomain }, function (_, k) { return tsHwm_1.valueOf() + BigInt(k * timeSpanPerDomain) / BigInt(tc.samplesPerDomain); });
                                    data = timestamps.map(function (ts) { return Math.sin(0.0000000001 * Number(ts)); });
                                    _loop_2 = function (j) {
                                        var writer, dataDict;
                                        return __generator(this, function (_e) {
                                            switch (_e.label) {
                                                case 0:
                                                    writer = writers[j];
                                                    dataDict = {};
                                                    tc.channels[j].indexChannels.forEach(function (indexChannel) {
                                                        dataDict[indexChannel] = timestamps;
                                                    });
                                                    tc.channels[j].dataChannels.forEach(function (dataChannel) {
                                                        dataDict[dataChannel] = data;
                                                    });
                                                    return [4 /*yield*/, writer.write(dataDict)];
                                                case 1:
                                                    _e.sent();
                                                    if (!!tc.autoCommit) return [3 /*break*/, 3];
                                                    return [4 /*yield*/, writer.commit()];
                                                case 2:
                                                    _e.sent();
                                                    _e.label = 3;
                                                case 3: return [2 /*return*/];
                                            }
                                        });
                                    };
                                    j = 0;
                                    _d.label = 1;
                                case 1:
                                    if (!(j < writers.length)) return [3 /*break*/, 4];
                                    return [5 /*yield**/, _loop_2(j)];
                                case 2:
                                    _d.sent();
                                    _d.label = 3;
                                case 3:
                                    j++;
                                    return [3 /*break*/, 1];
                                case 4:
                                    tsHwm_1.add(new x_1.TimeSpan(timeSpanPerDomain + 1));
                                    return [2 /*return*/];
                            }
                        });
                    };
                    i = 0;
                    _c.label = 5;
                case 5:
                    if (!(i < tc.domains)) return [3 /*break*/, 8];
                    return [5 /*yield**/, _loop_1(i)];
                case 6:
                    _c.sent();
                    _c.label = 7;
                case 7:
                    i++;
                    return [3 /*break*/, 5];
                case 8: return [3 /*break*/, 14];
                case 9:
                    _i = 0, writers_1 = writers;
                    _c.label = 10;
                case 10:
                    if (!(_i < writers_1.length)) return [3 /*break*/, 13];
                    writer = writers_1[_i];
                    return [4 /*yield*/, writer.close()];
                case 11:
                    _c.sent();
                    _c.label = 12;
                case 12:
                    _i++;
                    return [3 /*break*/, 10];
                case 13: return [7 /*endfinally*/];
                case 14: return [2 /*return*/];
            }
        });
    });
}
function parseInput(argv) {
    var argvCounter = 2;
    var identifier = argv[argvCounter++];
    var numWriters = parseInt(argv[argvCounter++]);
    var domains = parseInt(argv[argvCounter++]);
    var samplesPerDomain = parseInt(argv[argvCounter++]);
    var timeRangeStart = BigInt(argv[argvCounter++]);
    var timeRangeEnd = BigInt(argv[argvCounter++]);
    var autoCommit = argv[argvCounter++] === "true";
    var indexPersistInterval = new x_1.TimeSpan(BigInt(argv[argvCounter++]));
    var writerMode = parseInt(argv[argvCounter++]);
    var numberOfChannelGroups = parseInt(argv[argvCounter++]);
    var channelGroups = [];
    for (var i = 0; i < numberOfChannelGroups; i++) {
        var numberOfIndex = parseInt(argv[argvCounter++]);
        var numberOfData = parseInt(argv[argvCounter++]);
        var indexChannels = argv.slice(argvCounter, argvCounter + numberOfIndex);
        argvCounter += numberOfIndex;
        var dataChannels = argv.slice(argvCounter, argvCounter + numberOfData);
        argvCounter += numberOfData;
        channelGroups.push(new IndexWriterGroup(indexChannels, dataChannels));
    }
    return {
        identifier: identifier,
        numWriters: numWriters,
        domains: domains,
        samplesPerDomain: samplesPerDomain,
        timeRange: new x_1.TimeRange(timeRangeStart, timeRangeEnd),
        autoCommit: autoCommit,
        indexPersistInterval: indexPersistInterval,
        writerMode: writerMode,
        channels: channelGroups,
    };
}
function main() {
    return __awaiter(this, void 0, void 0, function () {
        var tc;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    tc = parseInput(process.argv);
                    return [4 /*yield*/, writeTest(tc).catch(function (error) {
                            console.error(error);
                            client.close();
                            (0, process_1.exit)(1);
                        })];
                case 1:
                    _a.sent();
                    client.close();
                    return [2 /*return*/];
            }
        });
    });
}
await main();
