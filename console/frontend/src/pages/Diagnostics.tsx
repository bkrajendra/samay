import type { ReactNode } from "react";
import { CheckCircle2, AlertTriangle, XCircle } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { usePolling } from "@/lib/usePolling";
import type { DiagnosticCheck } from "@/lib/api";
import { cn } from "@/lib/utils";

const icon: Record<DiagnosticCheck["status"], ReactNode> = {
  ok: <CheckCircle2 className="h-5 w-5 text-[hsl(var(--status-ok))]" />,
  warn: <AlertTriangle className="h-5 w-5 text-[hsl(var(--status-warn))]" />,
  fail: <XCircle className="h-5 w-5 text-[hsl(var(--status-fail))]" />,
};

export function Diagnostics() {
  const { data, error, loading } = usePolling<DiagnosticCheck[]>(api.diagnostics, 10000);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Diagnostics</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {loading && !data && <p className="text-sm text-muted-foreground">Loading…</p>}
        {error && <p className="text-sm text-destructive">Failed to load diagnostics: {error.message}</p>}
        {data?.map((check) => (
          <div
            key={check.name}
            className={cn(
              "flex items-start gap-3 rounded-md border p-3",
              check.status === "fail" && "border-[hsl(var(--status-fail))]/40",
              check.status === "warn" && "border-[hsl(var(--status-warn))]/40",
            )}
          >
            {icon[check.status]}
            <div className="flex flex-col">
              <span className="text-sm font-medium">{check.name}</span>
              <span className="text-sm text-muted-foreground">{check.message}</span>
            </div>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
