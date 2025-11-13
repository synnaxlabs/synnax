//
// Created by Emiliano Bonilla on 11/11/25.
//

#pragma once
#include <memory>
#include <vector>

#include "x/cpp/telem/series.h"

#include "x/go/telem/x/go/telem/telem.pb.h"

namespace telem {
/// @brief A frame is a collection of series mapped to their corresponding channel
/// keys.
class Frame {
    /// @brief private copy constructor that deep copies the frame.
    Frame(const Frame &other);

public:
    /// @brief the channels in the frame.
    std::unique_ptr<std::vector<std::uint32_t>> channels;
    /// @brief the series in the frame.
    std::unique_ptr<std::vector<Series>> series;

    Frame() = default;

    /// @brief move constructor.
    Frame(Frame &&other) noexcept;

    /// @brief move assignment operator.
    Frame &operator=(Frame &&other) noexcept;

    /// @brief allocates a frame that can hold the given number of series.
    /// @param size the number of series to allocate space for.
    explicit Frame(size_t size);

    /// @brief constructs the frame from its protobuf representation.
    /// @param f the protobuf representation of the frame.
    explicit Frame(const PBFrame &f);

    /// @brief constructs a frame with a single channel and series.
    /// @param chan the channel key corresponding to the given series.
    /// @param ser the series to add to the frame.
    Frame(const std::uint32_t &chan, telem::Series &&ser);

    explicit Frame(
        std::unordered_map<std::uint32_t, telem::SampleValue> &data,
        size_t cap = 0
    );

    /// @brief binds the frame to the given protobuf representation.
    /// @param f the protobuf representation to bind to. This pb must be non-null.
    void to_proto(PBFrame *f) const;

    /// @brief adds a channel and series to the frame.
    /// @param chan the channel key to add.
    /// @param ser the series to add for the channel key.
    void add(const std::uint32_t &chan, telem::Series &ser) const;

    /// @brief adds the given series to the frame for the given channel key. Unlike
    /// add,
    ///  this method moves the series into the frame, rather than copying it.
    /// @param chan the channel key to add.
    /// @param ser the series to add for the channel key.
    void emplace(const std::uint32_t &chan, telem::Series &&ser) const;

    /// @brief returns true if the frame has no series.
    [[nodiscard]] bool empty() const;

    friend std::ostream &operator<<(std::ostream &os, const Frame &f);

    /// @brief returns the sample for the given channel and index.
    template<typename NumericType>
    NumericType at(const std::uint32_t &key, const int &index) const {
        for (size_t i = 0; i < channels->size(); i++)
            if (channels->at(i) == key) return series->at(i).at<NumericType>(index);
        throw std::runtime_error("channel not found");
    }

    [[nodiscard]] telem::SampleValue at(const std::uint32_t &key, const int &index) const;

    /// @brief returns the number of series in the frame.
    [[nodiscard]] size_t size() const { return series != nullptr ? series->size() : 0; }

    [[nodiscard]] size_t length() const {
        if (series == nullptr || series->empty()) return 0;
        return series->at(0).size();
    }

    [[nodiscard]] bool contains(const std::uint32_t &key) const {
        return std::find(channels->begin(), channels->end(), key) != channels->end();
    }

    /// @brief returns the number of channel-series pairs that the frame can hold
    /// before resizing.
    [[nodiscard]] size_t capacity() const {
        return channels != nullptr ? channels->capacity() : 0;
    }

    /// @brief clears the frame of all channels and series, making it empty for
    /// reuse.
    void clear() const;

    /// @brief reserves the given number of series in the frame.
    void reserve(const size_t &size);

    /// @brief deep copies the frame, all of its series, and their data. This
    /// function must be used explicitly (instead of through a copy constructor) to
    /// avoid unintentional deep copies.
    [[nodiscard]] Frame deep_copy() const;

    /// @brief implements iterator support for the frame, allowing the caller to
    /// traverse the channel keys and series in the frame.
    struct Iterator {
        using iterator_category = std::forward_iterator_tag;
        using value_type = std::pair<std::uint32_t, telem::Series &>;
        using difference_type = std::ptrdiff_t;
        using pointer = value_type *;
        using reference = value_type &;

        Iterator(
            std::vector<std::uint32_t> &channels_ref,
            std::vector<telem::Series> &series_ref,
            const size_t pos
        ):
            channels(channels_ref), series(series_ref), pos(pos) {}

        value_type operator*() const { return {channels.at(pos), series.at(pos)}; }

        Iterator &operator++() {
            pos++;
            return *this;
        }

        bool operator!=(const Iterator &other) const { return pos != other.pos; }

        bool operator==(const Iterator &other) const { return pos == other.pos; }

    private:
        std::vector<std::uint32_t> &channels;
        std::vector<telem::Series> &series;
        size_t pos;
    };

    [[nodiscard]] Iterator begin() const { return {*channels, *series, 0}; }

    [[nodiscard]] Iterator end() const {
        return {*channels, *series, channels->size()};
    }
};
};