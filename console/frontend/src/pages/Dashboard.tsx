import type { ReactNode } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ConfirmButton } from "@/components/ConfirmButton";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import { usePolling } from "@/lib/usePolling";
import { formatDuration, formatOffset, formatPpm, formatAgo } from "@/lib/format";

function Stat({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-sm font-medium">{value}</span>
    </div>
  );
}

export function Dashboard() {
  const status = usePolling(api.status, 7000);
  const tracking = usePolling(api.tracking, 7000);

  const running = status.data?.running ?? false;
  const synchronized = status.data?.synchronized ?? false;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center gap-3">
        <Badge variant={running ? "ok" : "fail"}>chronyd: {running ? "Running" : "Stopped"}</Badge>
        <Badge variant={synchronized ? "ok" : "warn"}>
          {synchronized ? "Synchronized" : "Not synchronized"}
        </Badge>
        {status.data && (
          <span className="text-sm text-muted-foreground">
            {status.data.sourceCount} sources ({status.data.reachableSourceCount} reachable) ·{" "}
            {status.data.clientCount} clients
          </span>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Time</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <Stat label="Server time" value={tracking.data ? new Date(tracking.data.serverTimeLocal).toLocaleString() : "—"} />
          <Stat label="UTC time" value={tracking.data ? new Date(tracking.data.serverTimeUtc).toUTCString() : "—"} />
          <Stat label="Timezone" value={tracking.data?.timezone ?? "—"} />
          <Stat
            label="Last successful sync"
            value={tracking.data?.lastSyncAgoSecs !== undefined ? formatAgo(tracking.data.lastSyncAgoSecs) : "—"}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Synchronization</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <Stat label="Stratum" value={tracking.data?.stratum ?? "—"} />
          <Stat label="Reference source" value={tracking.data ? `${tracking.data.refName || tracking.data.refId}` : "—"} />
          <Stat label="Offset" value={tracking.data ? formatOffset(tracking.data.systemOffsetSecs) : "—"} />
          <Stat label="Frequency correction" value={tracking.data ? formatPpm(tracking.data.frequencyPpm) : "—"} />
          <Stat label="Root delay" value={tracking.data ? formatDuration(tracking.data.rootDelaySecs) : "—"} />
          <Stat label="Root dispersion" value={tracking.data ? formatDuration(tracking.data.rootDispersionSecs) : "—"} />
          <Stat label="Leap status" value={tracking.data?.leapStatus ?? "—"} />
          <Stat label="Update interval" value={tracking.data ? `${tracking.data.updateIntervalSecs.toFixed(1)}s` : "—"} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Service Operations</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-3">
          <Button variant="outline" onClick={() => api.sync()}>
            Force Sync
          </Button>
          <ConfirmButton
            title="Step the clock?"
            description="This immediately jumps the system clock to chronyd's current best estimate, instead of gradually slewing it. Anything relying on smooth, monotonic time may briefly see a jump."
            confirmLabel="Step clock"
            variant="outline"
            onConfirm={() => api.step()}
          >
            Step Clock
          </ConfirmButton>
          <ConfirmButton
            title="Restart chronyd?"
            description="This stops and restarts the chronyd process. Time serving on UDP 123 will briefly drop while it restarts."
            confirmLabel="Restart"
            variant="destructive"
            onConfirm={() => api.restart()}
          >
            Restart chronyd
          </ConfirmButton>
        </CardContent>
      </Card>
    </div>
  );
}
