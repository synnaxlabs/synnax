//
// Created by Emiliano Bonilla on 10/8/23.
//

#include "../telem.h"
#include <string>

using ChannelKey = std::uint32_t;

class Channel {
public:
    DataType dataType;
    cons std::string name;
    ChannelKey key;
    ChannelKey index;
    Rate rate;
    bool is_index;
    std::uint32_t leaseholder;

    Channel(
            const std::string name,
            DataType dataType,
            Rate rate = Rate(0),
            bool is_index = false,
            std::uint32_t leaseholder = 0,
            ChannelKey index = 0,
            ChannelKey key = 0
    ) : name(name),
        dataType(dataType),
        rate(rate),
        is_index(is_index),
        leaseholder(leaseholder),
        index(index),
        key(key) {}
};
