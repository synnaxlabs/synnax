#include <condition_variable>
#include <mutex>
#include <queue>

namespace queue {
template<typename T>
class SPSC {
public:
    void push(T value) {
        {
            std::lock_guard lock(mtx_);
            q_.push(std::move(value));
        }
        cv_.notify_one();
    }

    void pop(T &value) {
        std::unique_lock lock(mtx_);
        cv_.wait(lock, [&] { return !q_.empty(); });
        value = std::move(q_.front());
        q_.pop();
    }

    bool try_pop(T &value) {
        std::lock_guard lock(mtx_);
        if (q_.empty()) return false;
        value = std::move(q_.front());
        q_.pop();
        return true;
    }

    bool empty() const {
        std::lock_guard lock(mtx_);
        return q_.empty();
    }

private:
    mutable std::mutex mtx_;
    std::queue<T> q_;
    std::condition_variable cv_;
};
};
