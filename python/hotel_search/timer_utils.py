import time

# Global storage for start times
_timers = {}


def timer_start(label: str):
    """Starts a timer for a specific label."""
    _timers[label] = time.perf_counter()


def timer_end(label: str) -> float:
    """Returns the elapsed time for a label. Returns 0 if label not found."""
    start_time = _timers.get(label)
    if start_time is None:
        print(f"Warning: Timer '{label}' was never started.")
        return 0.0

    elapsed = time.perf_counter() - start_time
    return elapsed


def print_timer(label: str):
    """Convenience method to end a timer and print it immediately."""
    elapsed = timer_end(label)
    print(f"⏱️  {label}: {elapsed:.4f} seconds")
