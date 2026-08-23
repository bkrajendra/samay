import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api } from "@/lib/api";
import { usePolling } from "@/lib/usePolling";
import { formatOffset, formatDuration, formatAgo, reachToOctal } from "@/lib/format";
import type { SourceView } from "@/lib/api";

const statusVariant: Record<string, "ok" | "warn" | "fail" | "secondary"> = {
  Selected: "ok",
  Candidate: "secondary",
  NotCombined: "secondary",
  MayBeInError: "warn",
  TooVariable: "warn",
  Unusable: "fail",
  Unreachable: "fail",
};

export function Sources() {
  const { data, error, loading } = usePolling<SourceView[]>(api.sources, 5000);

  return (
    <Card>
      <CardHeader>
        <CardTitle>NTP Sources</CardTitle>
      </CardHeader>
      <CardContent>
        {loading && !data && <p className="text-sm text-muted-foreground">Loading…</p>}
        {error && <p className="text-sm text-destructive">Failed to load sources: {error.message}</p>}
        {data && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Source</TableHead>
                <TableHead>Stratum</TableHead>
                <TableHead>Reach</TableHead>
                <TableHead>Last Rx</TableHead>
                <TableHead>Offset</TableHead>
                <TableHead>Jitter</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((src) => (
                <TableRow key={src.address}>
                  <TableCell className="font-medium">{src.address}</TableCell>
                  <TableCell>{src.stratum}</TableCell>
                  <TableCell>{reachToOctal(src.reach)}</TableCell>
                  <TableCell>{formatAgo(src.lastRxSecs)}</TableCell>
                  <TableCell>{formatOffset(src.offsetSecs)}</TableCell>
                  <TableCell>{formatDuration(src.jitterSecs)}</TableCell>
                  <TableCell>
                    <Badge variant={statusVariant[src.status] ?? "secondary"}>{src.status}</Badge>
                  </TableCell>
                </TableRow>
              ))}
              {data.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7} className="text-center text-muted-foreground">
                    No sources configured
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
