//
// Created by Emiliano Bonilla on 10/8/23.
//

#include <string>

class TimeStamp {
    long value;
public:
    TimeStamp(long value) : value(value) {}
};

class TimeRange {
    TimeStamp start;
    TimeStamp end;
};

class DataType {
    std::string value;


public:
    DataType(std::string value) : value(value) {}

};

const STRING = DataType("string");

class Rate {
public:
    Rate(int i) {

    }

    float value;
};

class TimeSpan {
public:
    long value;
};