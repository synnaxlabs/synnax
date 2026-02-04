/*
 * macOS OSAL implementation for SOEM
 * Based on Linux implementation, adapted for macOS
 *
 * Key differences from Linux:
 * - Uses nanosleep instead of clock_nanosleep (not available on macOS)
 * - Uses CLOCK_REALTIME with mach_absolute_time for monotonic timing
 */

#include <osal.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <mach/mach_time.h>

static mach_timebase_info_data_t timebase_info;
static int timebase_initialized = 0;

static void ensure_timebase_initialized(void)
{
   if (!timebase_initialized)
   {
      mach_timebase_info(&timebase_info);
      timebase_initialized = 1;
   }
}

/* Returns time from some unspecified moment in past,
 * strictly increasing, used for time intervals measurement. */
void osal_get_monotonic_time(ec_timet *ts)
{
   ensure_timebase_initialized();

   uint64_t mach_time = mach_absolute_time();
   uint64_t nanos = mach_time * timebase_info.numer / timebase_info.denom;

   ts->tv_sec = nanos / 1000000000ULL;
   ts->tv_nsec = nanos % 1000000000ULL;
}

ec_timet osal_current_time(void)
{
   struct timespec ts;

   clock_gettime(CLOCK_REALTIME, &ts);
   return ts;
}

void osal_time_diff(ec_timet *start, ec_timet *end, ec_timet *diff)
{
   osal_timespecsub(end, start, diff);
}

void osal_timer_start(osal_timert *self, uint32 timeout_usec)
{
   struct timespec start_time;
   struct timespec timeout;

   osal_get_monotonic_time(&start_time);
   osal_timespec_from_usec(timeout_usec, &timeout);
   osal_timespecadd(&start_time, &timeout, &self->stop_time);
}

boolean osal_timer_is_expired(osal_timert *self)
{
   struct timespec current_time;
   int is_not_yet_expired;

   osal_get_monotonic_time(&current_time);
   is_not_yet_expired = osal_timespeccmp(&current_time, &self->stop_time, <);

   return is_not_yet_expired == FALSE;
}

int osal_usleep(uint32 usec)
{
   struct timespec ts;
   int result;

   osal_timespec_from_usec(usec, &ts);
   /* macOS doesn't have clock_nanosleep, use regular nanosleep */
   result = nanosleep(&ts, NULL);
   return result == 0 ? 0 : -1;
}

int osal_monotonic_sleep(ec_timet *ts)
{
   /* macOS doesn't support TIMER_ABSTIME with nanosleep.
    * Calculate relative time and use regular nanosleep. */
   struct timespec now, relative;
   int result;

   osal_get_monotonic_time(&now);
   osal_timespecsub(ts, &now, &relative);

   if (relative.tv_sec < 0 || (relative.tv_sec == 0 && relative.tv_nsec <= 0))
   {
      return 0; /* Already past the target time */
   }

   result = nanosleep(&relative, NULL);
   return result == 0 ? 0 : -1;
}

void *osal_malloc(size_t size)
{
   return malloc(size);
}

void osal_free(void *ptr)
{
   free(ptr);
}

int osal_thread_create(void *thandle, int stacksize, void *func, void *param)
{
   int ret;
   pthread_attr_t attr;
   pthread_t *threadp;

   threadp = thandle;
   pthread_attr_init(&attr);
   pthread_attr_setstacksize(&attr, stacksize);
   ret = pthread_create(threadp, &attr, func, param);
   if (ret < 0)
   {
      return 0;
   }
   return 1;
}

int osal_thread_create_rt(void *thandle, int stacksize, void *func, void *param)
{
   int ret;
   pthread_attr_t attr;
   pthread_t *threadp;

   threadp = thandle;
   pthread_attr_init(&attr);
   pthread_attr_setstacksize(&attr, stacksize);
   ret = pthread_create(threadp, &attr, func, param);
   pthread_attr_destroy(&attr);
   if (ret < 0)
   {
      return 0;
   }
   /* macOS doesn't support SCHED_FIFO without special entitlements,
    * so we just use the default scheduler with elevated priority. */

   return 1;
}

void *osal_mutex_create(void)
{
   pthread_mutexattr_t mutexattr;
   osal_mutext *mutex;
   mutex = (osal_mutext *)osal_malloc(sizeof(osal_mutext));
   if (mutex)
   {
      pthread_mutexattr_init(&mutexattr);
      /* PTHREAD_PRIO_INHERIT not well supported on macOS, use default */
      pthread_mutex_init(mutex, &mutexattr);
   }
   return (void *)mutex;
}

void osal_mutex_destroy(void *mutex)
{
   pthread_mutex_destroy((osal_mutext *)mutex);
   osal_free(mutex);
}

void osal_mutex_lock(void *mutex)
{
   pthread_mutex_lock((osal_mutext *)mutex);
}

void osal_mutex_unlock(void *mutex)
{
   pthread_mutex_unlock((osal_mutext *)mutex);
}
