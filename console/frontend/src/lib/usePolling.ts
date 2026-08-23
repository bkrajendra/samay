import * as React from "react";

/**
 * Polls fetcher every intervalMs, exposing the latest data/error. Refetches
 * immediately on mount and whenever refreshKey changes.
 */
export function usePolling<T>(fetcher: () => Promise<T>, intervalMs = 7000) {
  const [data, setData] = React.useState<T | null>(null);
  const [error, setError] = React.useState<Error | null>(null);
  const [loading, setLoading] = React.useState(true);
  const fetcherRef = React.useRef(fetcher);
  fetcherRef.current = fetcher;

  const load = React.useCallback(async () => {
    try {
      const result = await fetcherRef.current();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    load();
    const id = setInterval(load, intervalMs);
    return () => clearInterval(id);
  }, [load, intervalMs]);

  return { data, error, loading, refresh: load };
}
