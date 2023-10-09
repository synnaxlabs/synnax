//
// Created by Emiliano Bonilla on 10/8/23.
//

#include <string>

class TimeStamp {
public:
    long value;

    TimeStamp(long value) : value(value) {}
};

class TimeRange {
public:
    TimeStamp end;
    TimeStamp start;

    TimeRange(TimeStamp start, TimeStamp end) : start(start), end(end) {}
};

class DataType {


public:
    explicit DataType(std::string value) : value(value) {}

    // copy constructor
    DataType(const DataType &obj) {
        value = obj.value;
    }

    std::string value;

};


const STRING = DataType("string");

class Rate {
public:
    explicit Rate(float i) : value(i) {
    }

    float value;
};

class TimeSpan {
public:
    long value;
};