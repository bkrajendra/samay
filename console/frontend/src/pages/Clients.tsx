import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api } from "@/lib/api";
import { usePolling } from "@/lib/usePolling";
import { formatAgo } from "@/lib/format";
import type { ClientView } from "@/lib/api";

const statusVariant: Record<string, "ok" | "warn" | "fail"> = {
  Active: "ok",
  Stale: "warn",
  Offline: "fail",
};

export function Clients() {
  const { data, error, loading } = usePolling<ClientView[]>(api.clients, 5000);

  return (
    <Card>
      <CardHeader>
        <CardTitle>NTP Clients</CardTitle>
      </CardHeader>
      <CardContent>
        {loading && !data && <p className="text-sm text-muted-foreground">Loading…</p>}
        {error && <p className="text-sm text-destructive">Failed to load clients: {error.message}</p>}
        {data && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Client</TableHead>
                <TableHead>Last Request</TableHead>
                <TableHead>Requests</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((c) => (
                <TableRow key={c.address}>
                  <TableCell className="font-medium">{c.address}</TableCell>
                  <TableCell>{formatAgo(c.lastRequestAgoSecs)}</TableCell>
                  <TableCell>{c.ntpRequests}</TableCell>
                  <TableCell>
                    <Badge variant={statusVariant[c.status] ?? "secondary"}>{c.status}</Badge>
                  </TableCell>
                </TableRow>
              ))}
              {data.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-muted-foreground">
                    No clients have contacted this server yet
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
