#pragma once

#include <vector>

// Freighter.
#include "freighter/gRPC/client.h"

//
#include "synnax/telem/telem.h"
#include "synnax/telem/series.h"
#include "synnax/channel/channel.h"

using namespace Synnax;

/// @brief Frame type.
class Frame {
    std::vector<ChannelKey> channels;
    std::vector<Telem::Series> series;
};