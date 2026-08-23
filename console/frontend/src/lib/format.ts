export function formatOffset(seconds: number): string {
  const abs = Math.abs(seconds);
  const sign = seconds < 0 ? "-" : "+";
  if (abs < 1e-6) return `${sign}${(abs * 1e9).toFixed(0)}ns`;
  if (abs < 1e-3) return `${sign}${(abs * 1e6).toFixed(2)}us`;
  if (abs < 1) return `${sign}${(abs * 1e3).toFixed(2)}ms`;
  return `${sign}${abs.toFixed(3)}s`;
}

export function formatDuration(seconds: number): string {
  const abs = Math.abs(seconds);
  if (abs < 1e-6) return `${(abs * 1e9).toFixed(0)}ns`;
  if (abs < 1e-3) return `${(abs * 1e6).toFixed(2)}us`;
  if (abs < 1) return `${(abs * 1e3).toFixed(2)}ms`;
  return `${abs.toFixed(3)}s`;
}

export function formatPpm(ppm: number): string {
  return `${ppm >= 0 ? "+" : ""}${ppm.toFixed(3)} ppm`;
}

export function formatAgo(seconds: number | null | undefined): string {
  if (seconds === null || seconds === undefined) return "never";
  if (seconds < 60) return `${Math.floor(seconds)}s ago`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  return `${Math.floor(seconds / 86400)}d ago`;
}

export function reachToOctal(reach: number): string {
  return reach.toString(8).padStart(3, "0");
}
