// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.

extern "C" {
#include "osal.h"
}

#include <atomic>
#include <thread>
#include <vector>

#include <gtest/gtest.h>
#include <pthread.h>

/// @brief Test fixture for OSAL time and timer tests.
class OsalTimeTest : public ::testing::Test {
protected:
    ec_timet ts1;
    ec_timet ts2;
    ec_timet diff;
    osal_timert timer;
};

/// @brief Test fixture for OSAL threading tests.
class OsalThreadTest : public ::testing::Test {
protected:
    pthread_t handle;
    std::atomic<int> counter{0};
};

/// @brief Test fixture for OSAL mutex tests.
class OsalMutexTest : public ::testing::Test {
protected:
    void *mutex;

    void SetUp() override { this->mutex = osal_mutex_create(); }

    void TearDown() override {
        if (this->mutex != nullptr) osal_mutex_destroy(this->mutex);
    }
};

TEST_F(OsalTimeTest, MonotonicTimeStrictlyIncreases) {
    osal_get_monotonic_time(&this->ts1);
    osal_usleep(1000);
    osal_get_monotonic_time(&this->ts2);

    bool second_is_greater = (this->ts2.tv_sec > this->ts1.tv_sec) ||
                             (this->ts2.tv_sec == this->ts1.tv_sec &&
                              this->ts2.tv_nsec > this->ts1.tv_nsec);
    EXPECT_TRUE(second_is_greater);
}

TEST_F(OsalTimeTest, CurrentTimeReturnsValidTime) {
    ec_timet current = osal_current_time();
    EXPECT_GT(current.tv_sec, 1700000000);
    EXPECT_LT(current.tv_sec, 2000000000);
}

TEST_F(OsalTimeTest, TimeDiffCalculatesCorrectly) {
    this->ts1.tv_sec = 10;
    this->ts1.tv_nsec = 500000000;
    this->ts2.tv_sec = 12;
    this->ts2.tv_nsec = 300000000;

    osal_time_diff(&this->ts1, &this->ts2, &this->diff);

    EXPECT_EQ(this->diff.tv_sec, 1);
    EXPECT_EQ(this->diff.tv_nsec, 800000000);
}

TEST_F(OsalTimeTest, TimerNotExpiredImmediately) {
    osal_timer_start(&this->timer, 10000);
    EXPECT_FALSE(osal_timer_is_expired(&this->timer));
}

TEST_F(OsalTimeTest, TimerExpiresAfterTimeout) {
    osal_timer_start(&this->timer, 5000);
    osal_usleep(10000);
    EXPECT_TRUE(osal_timer_is_expired(&this->timer));
}

TEST_F(OsalTimeTest, TimerWithZeroTimeoutExpiresImmediately) {
    osal_timer_start(&this->timer, 0);
    EXPECT_TRUE(osal_timer_is_expired(&this->timer));
}

TEST_F(OsalTimeTest, UsleepReturnsZeroOnSuccess) {
    int result = osal_usleep(1000);
    EXPECT_EQ(result, 0);
}

TEST_F(OsalTimeTest, UsleepDurationWithinTolerance) {
    osal_get_monotonic_time(&this->ts1);
    osal_usleep(10000);
    osal_get_monotonic_time(&this->ts2);

    osal_time_diff(&this->ts1, &this->ts2, &this->diff);

    int64_t elapsed_usec = this->diff.tv_sec * 1000000 + this->diff.tv_nsec / 1000;
    EXPECT_GE(elapsed_usec, 10000);
    EXPECT_LE(elapsed_usec, 30000);
}

TEST_F(OsalTimeTest, MonotonicSleepUntilFutureTime) {
    ec_timet target;
    osal_get_monotonic_time(&target);
    target.tv_nsec += 10000000;
    if (target.tv_nsec >= 1000000000) {
        target.tv_sec += 1;
        target.tv_nsec -= 1000000000;
    }

    int result = osal_monotonic_sleep(&target);
    EXPECT_EQ(result, 0);

    osal_get_monotonic_time(&this->ts1);
    bool reached_target = (this->ts1.tv_sec > target.tv_sec) ||
                          (this->ts1.tv_sec == target.tv_sec &&
                           this->ts1.tv_nsec >= target.tv_nsec);
    EXPECT_TRUE(reached_target);
}

TEST(OsalMemoryTest, MallocReturnsNonNull) {
    void *ptr = osal_malloc(1024);
    EXPECT_NE(ptr, nullptr);
    osal_free(ptr);
}

TEST(OsalMemoryTest, MallocAndFreeDoNotCrash) {
    void *ptr = osal_malloc(4096);
    ASSERT_NE(ptr, nullptr);
    osal_free(ptr);
}

static void *thread_increment_counter(void *param) {
    std::atomic<int> *counter = static_cast<std::atomic<int> *>(param);
    counter->fetch_add(1);
    return nullptr;
}

TEST_F(OsalThreadTest, ThreadCreateReturnsOneOnSuccess) {
    int result = osal_thread_create(
        &this->handle,
        65536,
        reinterpret_cast<void *>(thread_increment_counter),
        &this->counter
    );
    EXPECT_EQ(result, 1);
    pthread_join(this->handle, nullptr);
}

TEST_F(OsalThreadTest, ThreadExecutesFunction) {
    this->counter.store(0);
    int result = osal_thread_create(
        &this->handle,
        65536,
        reinterpret_cast<void *>(thread_increment_counter),
        &this->counter
    );
    ASSERT_EQ(result, 1);
    pthread_join(this->handle, nullptr);
    EXPECT_EQ(this->counter.load(), 1);
}

TEST_F(OsalThreadTest, ThreadCreateRtReturnsOneOnSuccess) {
    int result = osal_thread_create_rt(
        &this->handle,
        65536,
        reinterpret_cast<void *>(thread_increment_counter),
        &this->counter
    );
    EXPECT_EQ(result, 1);
    pthread_join(this->handle, nullptr);
}

TEST_F(OsalMutexTest, MutexCreateReturnsNonNull) {
    EXPECT_NE(this->mutex, nullptr);
}

TEST_F(OsalMutexTest, MutexLockUnlockDoNotCrash) {
    ASSERT_NE(this->mutex, nullptr);
    osal_mutex_lock(this->mutex);
    osal_mutex_unlock(this->mutex);
}

struct MutexTestContext {
    void *mutex;
    std::atomic<int> *counter;
    int iterations;
};

static void *mutex_increment_worker(void *param) {
    MutexTestContext *ctx = static_cast<MutexTestContext *>(param);
    for (int i = 0; i < ctx->iterations; i++) {
        osal_mutex_lock(ctx->mutex);
        int val = ctx->counter->load();
        ctx->counter->store(val + 1);
        osal_mutex_unlock(ctx->mutex);
    }
    return nullptr;
}

TEST_F(OsalMutexTest, MutexProvidesMutualExclusion) {
    ASSERT_NE(this->mutex, nullptr);

    const int num_threads = 10;
    const int iterations_per_thread = 1000;

    std::atomic<int> shared_counter{0};
    MutexTestContext ctx{this->mutex, &shared_counter, iterations_per_thread};

    std::vector<pthread_t> threads(num_threads);
    for (int i = 0; i < num_threads; i++) {
        int result = osal_thread_create(
            &threads[i],
            65536,
            reinterpret_cast<void *>(mutex_increment_worker),
            &ctx
        );
        ASSERT_EQ(result, 1);
    }

    for (int i = 0; i < num_threads; i++) {
        pthread_join(threads[i], nullptr);
    }

    EXPECT_EQ(shared_counter.load(), num_threads * iterations_per_thread);
}
